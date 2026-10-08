// A minimal TCP echo server, standard library only.
//
// What this demonstrates, and why it is built this way:
//
//  1. The three-way handshake is NOT here. By the time Accept() returns, the
//     kernel has already completed the handshake. Your code never sees a SYN.
//     Compare with net.ListenPacket / UDP, where there is no handshake at all.
//
//  2. Accept() returning is the first moment the connection exists as far as
//     your program is concerned. Everything about the handshake is kernel work.
//
//  3. The read/write asymmetry. TCP is a BYTE STREAM, not a message channel.
//     A client's three separate writes can arrive as one read, and one client's
//     write can arrive as three reads. This program prints how many bytes each
//     read returned, so you can watch that happen.
//
//  4. One goroutine per connection. The accept loop is the only place that
//     blocks on accept; each connection then blocks independently. This is the
//     cheapest concurrency model in the language and it maps exactly onto how
//     the kernel already models sockets.
//
//  5. Nagle's algorithm is off. Go sets TCP_NODELAY by default. If you build the
//     same server in C, small writes get buffered until the previous ones are
//     acknowledged, which adds a round trip of latency per small message.
//
// Run:
//
//	go run main.go              # listens on :8080, all interfaces
//	go run main.go 9000         # listens on :9000        (bare port argument)
//	go run main.go -port 9000   # same thing, explicit flag
//	go run main.go -host 127.0.0.1   # loopback only
//
// Then in another terminal:
//
//	nc 127.0.0.1 8080
//	hello
//	world
//
// Or from a shell without nc:
//
//	printf 'hello\nworld\n' | nc 127.0.0.1 8080
//
// Watch the socket state from a third terminal while connecting:
//
//	ss -tn state syn-sent
//
// To see the read/write asymmetry clearly, connect and type slowly: one line at
// a time, waiting for each echo. Then compare with piping all input at once.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

func main() {
	port := flag.Int("port", 8080, "TCP port to listen on")
	host := flag.String("host", "", "address to bind (empty means all interfaces)")
	flag.Parse()

	// Also accept a bare positional port, so both of these work:
	//   go run main.go -port 9000
	//   go run main.go 9000
	// flag.Args() holds everything the flag package did not consume. Without
	// this check a positional argument would be silently ignored, which is a
	// genuinely confusing failure: the server would start on 8080 anyway and
	// your client would connect to nothing.
	if flag.NArg() > 0 {
		p, err := strconv.Atoi(flag.Arg(0))
		if err != nil {
			log.Fatalf("port %q is not a number: %v", flag.Arg(0), err)
		}
		*port = p
	}

	// ":8080" binds all interfaces; "127.0.0.1:8080" binds loopback only.
	addr := net.JoinHostPort(*host, strconv.Itoa(*port))

	// net.Listen creates the listening socket and puts it in the LISTEN state.
	// No handshake happens here. The kernel is waiting for SYNs.
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen on %s: %v", addr, err)
	}
	defer ln.Close()

	fmt.Printf("listening on %s\n", ln.Addr())
	fmt.Printf("connect with:  nc 127.0.0.1 %d\n", *port)
	fmt.Println("stop with:     Ctrl-C")
	fmt.Println()

	// The accept loop. This is the ONLY sequential part of the server.
	// Everything interesting happens in handle(), concurrently.
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}

		// By the time we are here the handshake is DONE. The kernel did it.
		// spawn one goroutine per connection and go straight back to accept,
		// so a slow client cannot block new connections.
		go handle(conn)
	}
}

// handle services exactly one connection, then closes it.
func handle(conn net.Conn) {
	defer conn.Close()

	// RemoteAddr's port is the client's EPHEMERAL port, assigned by the client
	// kernel during the handshake. LocalAddr's port is what the client asked
	// for. This pair is the 4-tuple that identifies the connection, and it is
	// why a server can hold many connections to the same client IP.
	local := conn.LocalAddr().(*net.TCPAddr)
	remote := conn.RemoteAddr().(*net.TCPAddr)
	log.Printf("accepted %s:%d -> %s:%d", remote.IP, remote.Port, local.IP, local.Port)

	// bufio gives us ReadString, but note: it is reading from a BYTE STREAM.
	// ReadString finds a newline; it does not receive a "message".
	reader := bufio.NewReader(conn)

	// idleTimeout bounds how long we wait for the NEXT line. The deadline is
	// set inside the loop, not once here: SetDeadline is ABSOLUTE, so a single
	// call before the loop would kill the connection after 60 seconds of total
	// lifetime even if the client was actively using it. Re-arming it on every
	// iteration makes it an idle timeout, which is what we actually want.
	const idleTimeout = 60 * time.Second

	for {
		if err := conn.SetReadDeadline(time.Now().Add(idleTimeout)); err != nil {
			log.Printf("%s:%d set read deadline: %v", remote.IP, remote.Port, err)
			return
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			// io.EOF means the client closed its side cleanly (sent FIN).
			// A timeout means it just went quiet. Both end the connection,
			// but the cause is worth logging differently.
			if errors.Is(err, io.EOF) {
				log.Printf("%s:%d disconnected (client closed)", remote.IP, remote.Port)
			} else if ne, ok := err.(net.Error); ok && ne.Timeout() {
				log.Printf("%s:%d timed out after %s idle", remote.IP, remote.Port, idleTimeout)
			} else {
				log.Printf("%s:%d read: %v", remote.IP, remote.Port, err)
			}
			return
		}

		trimmed := strings.TrimRight(line, "\r\n")

		// This print is the whole lesson. "hello\nworld\n" typed quickly may
		// arrive as ONE read of 12 bytes, or as two reads of 6 bytes each, or
		// as three, depending on how the client's writes and the network
		// segment it up. The BYTES are identical either way.
		log.Printf("  read %d bytes: %q", len(line), trimmed)

		// Echo it back. This is an echo server: write what you read.
		//
		// Nagle's algorithm is OFF here. Go sets TCP_NODELAY by default on TCP
		// connections, because for request-response traffic a delayed small
		// write costs a full round trip and buys nothing. If you write 20 bytes
		// while an earlier write is still unacknowledged, Nagle would hold them
		// until the ACK arrived. Node's net module does the same thing by
		// default; most other runtimes leave Nagle enabled.
		if _, err := conn.Write([]byte(line)); err != nil {
			log.Printf("%s:%d write: %v", remote.IP, remote.Port, err)
			return
		}
	}
}
