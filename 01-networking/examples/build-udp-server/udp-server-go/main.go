// A minimal UDP echo server, standard library only.
//
// This is the deliberate contrast with the TCP server in this module. Every
// difference below traces back to one fact: UDP has no handshake, no connection,
// and no acknowledgements.
//
//  1. No Accept(). ReadFromUDP() blocks until a datagram arrives. There is no
//     LISTEN state, no SYN-RECV, nothing to accept.
//
//  2. No connection object. There is no Conn, no Close-on-shutdown subtlety, no
//     4-tuple. ReadFromUDP returns the sender's address with every datagram,
//     which is the only way you learn who sent it.
//
//  3. MESSAGE BOUNDARIES ARE PRESERVED. One datagram in, exactly one datagram
//     out — same bytes, never merged, never split. This is the sharpest
//     difference from TCP, where the bytes are identical but the chunking is
//     not observable.
//
//  4. No flow control and no congestion control. If the reader is slow,
//     ReadFromUDP's buffer fills and the kernel DROPS datagrams. There is no
//     retransmission and nobody is told. Increasing the read buffer is your only
//     lever, and it is finite. This is the practical cost of UDP that TCP hides.
//
// Run:
//
//	go run main.go              # listens on :9090, all interfaces
//	go run main.go 9000         # listens on :9000        (bare port argument)
//	go run main.go -port 9000   # same thing, explicit flag
//
// Then in another terminal:
//
//	nc -u 127.0.0.1 9090       # note the -u: UDP, not TCP
//	hello
//
// One-shot, no interactive session required:
//
//	echo -n 'hello' | nc -u -w1 127.0.0.1 9090
//
// See the burst case for what "no flow control" means in practice:
//
//	# send 5000 datagrams as fast as possible, count what comes back
//	for i in $(seq 5000); do echo -n "x" | nc -u -w1 127.0.0.1 9090 & done
package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync/atomic"
	"time"
)

// bufferSize is the socket receive buffer, in bytes. It is the ONLY knob you
// have against loss: if you cannot drain datagrams fast enough, the kernel
// drops them once this fills. 64 KB holds a lot of small datagrams; a
// high-rate server needs more.
const bufferSize = 64 * 1024

func main() {
	port := flag.Int("port", 9090, "UDP port to listen on")
	host := flag.String("host", "", "address to bind (empty means all interfaces)")
	flag.Parse()

	// Also accept a bare positional port, so both of these work:
	//   go run main.go -port 9000
	//   go run main.go 9000
	// flag.Args() holds everything the flag package did not consume. Without
	// this check a positional argument would be silently ignored and the server
	// would bind 9090 anyway, which is a baffling way to lose a connection.
	if flag.NArg() > 0 {
		p, err := strconv.Atoi(flag.Arg(0))
		if err != nil {
			log.Fatalf("port %q is not a number: %v", flag.Arg(0), err)
		}
		*port = p
	}

	addr := net.JoinHostPort(*host, strconv.Itoa(*port))

	// ListenUDP binds the socket. Compare net.Listen("tcp", ...) in the TCP
	// server: the type "udp" means there is no accept loop, no handshake, and
	// no connection state to create.
	conn, err := net.ListenUDP("udp", mustResolve(addr))
	if err != nil {
		log.Fatalf("listen udp %s: %v", addr, err)
	}
	defer conn.Close()

	// Ask the kernel for a bigger receive buffer. Best effort: if the system
	// caps it lower, ReadFromUDP still works, just with a smaller buffer.
	if err := conn.SetReadBuffer(bufferSize); err != nil {
		log.Printf("could not raise read buffer: %v", err)
	}

	fmt.Printf("listening on %s (udp)\n", conn.LocalAddr())
	fmt.Printf("connect with:  nc -u 127.0.0.1 %d\n", *port)
	fmt.Println("stop with:     Ctrl-C")
	fmt.Println()

	// A scratch buffer, reused across every datagram. UDP's Length field tells
	// you how long each datagram is, so one fixed buffer serves any message —
	// which is a luxury TCP does not offer, where you must append to a growing
	// buffer because you cannot know where a message ends.
	//
	// The buffer MUST be at least as large as the largest datagram you intend to
	// accept. ReadFromUDP does not error on an oversized datagram: it TRUNCATES
	// it and returns n == len(buf), with the excess silently discarded and
	// without ICMP. That is UDP's message-boundary guarantee working against you.
	//
	// The theoretical maximum is 65,507 bytes (a 16-bit IPv4 total-length field,
	// minus a 20-byte IP header, minus this 8-byte UDP header). Allocating that
	// per-read would be wasteful for an echo server handling small messages, so
	// this uses 2 KB and rejects anything larger, loudly. Raise it to 65507 only
	// if you genuinely need to accept maximum-size datagrams.
	const maxDatagram = 2048
	buf := make([]byte, maxDatagram)

	var received, bytesIn, bytesOut atomic.Uint64

	// Print a running tally so you can compare what you SENT against what
	// arrived. UDP will never report the difference itself — this is the only
	// way you would discover loss.
	go func() {
		t := time.NewTicker(time.Second)
		defer t.Stop()
		var prevRx, prevOut uint64
		for range t.C {
			rx, out := received.Load(), bytesOut.Load()
			if rx != prevRx || out != prevOut {
				fmt.Printf("[dgrams=%d  bytesIn=%d  echoed=%d]\n",
					rx, bytesIn.Load(), out)
				prevRx, prevOut = rx, out
			}
		}
	}()

	for {
		// THE call that replaces Accept(). It blocks until a datagram arrives,
		// then returns the data AND the sender's address in one shot.
		//
		// n is the real datagram length. A UDP datagram is delivered whole or
		// not at all — there is no partial read, no reassembly, no buffering of
		// a partial message. This is what makes message framing free here.
		n, from, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("read from %v: %v", from, err)
			continue
		}

		// Detect truncation. ReadFromUDP reports n == len(buf) when it had to
		// discard the tail, and does NOT send ICMP. Catching it here turns a
		// silent data loss into a visible log line — which is the only place you
		// can possibly notice it in UDP.
		if n == len(buf) {
			log.Printf("  WARNING: datagram from %s filled the %d-byte buffer; "+
				"it was truncated. Raise maxDatagram to accept it.", from, len(buf))
		}

		received.Add(1)
		bytesIn.Add(uint64(n))
		log.Printf("  %d bytes from %s: %q", n, from, buf[:n])

		// Echo to the address that just sent. There is no session to remember,
		// so this is stateless — which is exactly why UDP scales so easily and
		// why it is trivially spoofable.
		written, err := conn.WriteToUDP(buf[:n], from)
		if err != nil {
			log.Printf("write to %v: %v", from, err)
			continue
		}
		bytesOut.Add(uint64(written))
	}
}

func mustResolve(addr string) *net.UDPAddr {
	a, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		log.Fatalf("resolve %s: %v", addr, err)
	}
	return a
}
