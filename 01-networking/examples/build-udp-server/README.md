# Build a UDP echo server

Runnable echo servers in Go and TypeScript, built to contrast directly with the
TCP servers one directory over.

Related note: [05-udp.md](../../05-udp.md)

## Layout

```text
build-udp-server/
├── udp-server-go/main.go            # net.ListenUDP + ReadFromUDP loop
├── udp-server-typescript/main.ts    # dgram.createSocket + one 'message' handler
└── README.md                        # this file
```

## What is missing compared to the TCP server

Read the two directories side by side. Almost everything absent here is absent
*because UDP has no handshake*:

| | TCP example | UDP example |
|---|---|---|
| Bind | `net.Listen` | `net.ListenUDP` |
| Accept something | `ln.Accept()` — blocks per connection | **nothing to accept** |
| Connection object | `net.Conn` | **none** |
| Peer identity | `conn.RemoteAddr()`, stable | returned per datagram, port not stable |
| Concurrency unit | one goroutine / one socket | **one handler for everyone** |
| Shutdown | drain connections, then close | `close()` and done |
| Byte stream | yes, boundaries lost | no, **boundaries preserved** |
| Loss | invisible, retransmitted | **silent and permanent** |

The Go version's `ReadFromUDP` loop replaces the TCP version's accept loop. That
one substitution is most of the difference.

## Prerequisites

| Example | Requires | Verified on |
|---|---|---|
| Go | Go 1.22+ (standard library only) | go1.27.1 linux/amd64 |
| TypeScript | Node 20.6+ with `--experimental-strip-types`, Node 22.6+, or Bun 1.x | Node v26.7.0, Bun 1.4.2 |

The Go example needs nothing — one file, standard library only, no `go mod init`.

The TypeScript example imports only `node:dgram`. Its `package.json` and
`tsconfig.json` exist **solely to typecheck**:

```bash
cd udp-server-typescript
bun install        # devDependencies, for typechecking only
bun run typecheck  # must exit 0
```

There is **no build step**. See
[../build-tcp-server/README.md](../build-tcp-server/README.md#type-safety) for
what each strictness flag is doing there; the config here is identical.

## Running the Go version

```bash
cd udp-server-go
go run main.go                   # listens on :9090, all interfaces
go run main.go 9000              # listens on :9000        (bare port argument)
go run main.go -port 9000        # same thing, explicit flag
go run main.go -host 127.0.0.1   # loopback only
```

```text
listening on [::]:9090 (udp)
connect with:  nc -u 127.0.0.1 9090
stop with:     Ctrl-C

2026/10/07 23:03:26   5 bytes from 127.0.0.1:37286: "hello"
[dgrams=1  bytesIn=5  echoed=5]
```

Real output from a verified run.

## Running the TypeScript version

```bash
cd udp-server-typescript

# Node 20.6+ (or 22.6+ where the flag is no longer needed)
node --experimental-strip-types main.ts 9090
npm start -- 9090

# Bun, no flag needed
bun main.ts 9090
bun main.ts --port 9090            # a bare port, --port N, and --port=N all work
```

Verified output:

```text
$ bun main.ts 9090
listening on :9090 (udp4)
connect with:  nc -u 127.0.0.1 9090
stop with:     Ctrl-C

  1 bytes from 127.0.0.1:48095: "a"
[dgrams=1  bytesIn=1  echoed=1]
  2 bytes from 127.0.0.1:58532: "bb"
[dgrams=2  bytesIn=3  echoed=3]
  3 bytes from 127.0.0.1:38590: "ccc"
[dgrams=3  bytesIn=6  echoed=6]
```

```text
listening on :9090 (udp4)
connect with:  nc -u 127.0.0.1 9090
stop with:     Ctrl-C

  1 bytes from 127.0.0.1:48095: "a"
[dgrams=1  bytesIn=1  echoed=1]
```

## Connecting — the `-u` matters

```bash
nc -u 127.0.0.1 9090       # the -u is essential
nc 127.0.0.1 9090          # TCP: connects, sends nothing back, looks broken
```

Without `-u`, `nc` speaks TCP. The TCP connection will succeed (nothing is
listening, but UDP has no handshake to refuse it) and you will receive silence.
This is a genuinely confusing first experience with UDP and worth doing once on
purpose.

One-shot, no interactive session:

```bash
echo -n 'hello' | nc -u -w1 127.0.0.1 9090
```

## What to observe

### 1. Message boundaries are preserved

Send three datagrams of 1, 2, and 3 bytes:

```text
2026/10/07 23:03:27   1 bytes from 127.0.0.1:42462: "a"
2026/10/07 23:03:28   2 bytes from 127.0.0.1:60096: "bb"
2026/10/07 23:03:29   3 bytes from 127.0.0.1:44178: "ccc"
```

One send, one receive, exactly the same length, three times. Compare with the
TCP example, where twelve bytes arrived as either one 12-byte read or two 3-byte
reads depending on timing.

This is why one fixed `make([]byte, 1024)` buffer is sufficient here. With TCP
you must append to a growing buffer precisely because you cannot know where a
message ends. **UDP's one weakness — no reliability — buys you message framing
for free.**

### 2. There is no handshake, so there is no refusal

Start the server, then connect with plain TCP `nc`:

```bash
nc 127.0.0.1 9090
```

The connection succeeds and nothing comes back. The TCP server refuses
instantly with a RST because nothing is listening on a TCP port. UDP has no
concept of refusal at the socket layer, which is a large part of why UDP is the
preferred vector for amplification attacks (RFC 5357).

### 3. The sender's port is not stable

Look at the source ports in the log above: `42462`, `60096`, `44178`. Each
`nc` invocation is a new client with a new ephemeral port. The server is
completely stateless — it has no table of peers.

This is exactly why a DNS client must explicitly bind a random port before
sending a query. It needs a stable port in order to match the reply back to the
query, and with no handshake to assign one, it must choose one itself. The
TypeScript example's header comment makes this point at `bind()`.

### 4. Loss is silent

Send a burst and count the replies:

```bash
for i in $(seq 2000); do echo -n "x" | nc -u -w1 127.0.0.1 9090; done | wc -l
```

The count is often **less than 2000**. Nothing reports an error, no socket
returns a failure, and the sender is never told. If the handler is busy when
datagrams arrive, the kernel's receive buffer fills and datagrams are dropped.

Both servers print a running `[dgrams=... bytesIn=... echoed=...]` tally
specifically so you can compare what you sent against what arrived. **That
tally is the only feedback you will ever get** — there is no ACK to inspect, no
`CLOSE_WAIT` to observe, no retransmission counter.

### 5. Oversized datagrams are truncated *silently*

This is the sharpest trap in UDP, and it is the exact inverse of point 1. Because
message boundaries are preserved, the receiver must know the whole message up
front — so the read buffer has to be at least as large as the largest datagram
you accept. **If it is not, `ReadFromUDP` does not return an error. It truncates,
and discards the rest without telling anyone.**

The Go example uses a 2048-byte buffer and detects this:

```bash
python3 -c "import sys; sys.stdout.write('X'*4000)" \
  | nc -u -w1 127.0.0.1 9090 > out.txt
wc -c < out.txt        # 2048 — you sent 4000
```

```text
2026/10/07 23:23:14   WARNING: datagram from 127.0.0.1:35825 filled the 2048-byte buffer; it was truncated. Raise maxDatagram to accept it.
2026/10/07 23:23:14   2048 bytes from 127.0.0.1:35825: "XXXX..."
```

Real output from a verified run. Four thousand bytes in, 2048 out, and **UDP
itself reported no error of any kind** — no ICMP, no `ECONNREFUSED`, nothing.
The sender received what looks like a successful reply.

The theoretical maximum datagram is 65,507 bytes: a 16-bit IPv4 total-length
field (65,535) minus a 20-byte IP header minus the 8-byte UDP header. If you must
accept maximum-size datagrams, size your buffer for 65,507 and accept the memory
cost.

### 6. Raising the receive buffer is your only lever

```go
conn.SetReadBuffer(bufferSize)   # 64 KB in the Go example
```

There is no flow control to negotiate, so the receive buffer is the only defense
against loss. It is finite, which is the practical limit. `SetReadBuffer` is
best-effort — the system may cap it lower, which is why the Go example logs a
warning if the call fails rather than assuming success.

This is a **different buffer** from point 5, and conflating them is easy. The
64 KB here is the *kernel socket receive buffer*, which holds many datagrams
waiting to be read. The 2048 bytes in point 5 is the *single-datagram read
buffer* your code passes into `ReadFromUDP`. Both must be right, for unrelated
reasons, and fixing one does nothing for the other.

### 7. UDP is trivially spoofable

Both servers reply to whatever address the datagram claims to come from, using
`WriteToUDP(buf, from)`. Nothing verifies that the sender actually owns that
address. A forged source address makes the reply go to someone else — the
mechanism behind spoofed traffic and reflection attacks. TCP's handshake makes
this much harder, because the attacker must receive a SYN-ACK at an address they
control.

## Things to change

- **Exhaust the receive buffer.** Add a `time.Sleep(100 * time.Millisecond)` at
  the top of the handler and rerun the burst test. Loss should increase sharply,
  and the tally will show it.
- **Add a sequence number to each datagram.** Send 1–2000 and identify exactly
  which numbers went missing. This is how you would build an application-level
  reliability layer, and it shows how much work UDP delegates to you.
- **Bind a specific source port.** Change the client to bind port 0 explicitly
  and watch the source port stop changing between requests.
- **Test the size limit.** Send a datagram larger than 65,507 bytes and read the
  error. Then work out the arithmetic: a 16-bit IPv4 total-length field minus a
  20-byte header minus the 8-byte UDP header.
- **Verify the checksum is mandatory in IPv6.** Run the server on IPv6
  (`udp6` / `tcp6` address family) and inspect a captured datagram — the
  checksum field cannot be zero.

## Related

- [05-udp.md](../../05-udp.md) — the note these examples demonstrate.
- [../build-tcp-server/](../build-tcp-server/) — the same servers with a
  handshake, for contrast.
