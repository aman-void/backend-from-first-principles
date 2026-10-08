---
title: TCP
module: 01-networking
status: draft
prerequisites:
  - ./01-all-about-the-internet.md
  - ./02-how-data-transfers.md
---

# TCP

TCP takes the internet's unreliable, unordered packet delivery and turns it into
an ordered, complete, error-checked byte stream between two processes. It is the
reason a program can `write()` 3 MB and then `read()` exactly those 3 MB, in
order, without knowing anything about routers, paths, or packet loss.

The critical thing to understand up front: **TCP provides none of this itself.**
It inherits the packet-switching model from the previous note — independent
packets, possibly different paths, loss and reordering as normal — and then adds
a bookkeeping scheme at each endpoint to detect and repair the difference between
what was sent and what arrived.

Every mechanism below exists to answer one of three questions:

1. **Did it arrive?** → sequence numbers, acknowledgements, checksums
2. **Did it arrive in order?** → sequence numbers, reassembly at the receiver
3. **How fast may I send?** → flow control (receiver) and congestion control
   (network)

## Why does this exist?

The previous note established that the internet core makes a best-effort promise:
it will try, it may drop your packet, it may reorder it, and it will not tell you
what capacity you will get. Applications cannot be built on that.

Consider what a naive program actually needs from a transport. It calls
`write("hello")` and later `read()`. For that to work:

- The five bytes must arrive.
- They must arrive **in the order** they were written, because the application
  has no concept of out-of-order delivery and would treat reordered data as
  corruption.
- The bytes must arrive **exactly once** — a duplicated byte stream is as broken
  as a truncated one.
- They must be **verified**, because a flipped bit somewhere in a fibre or a
  radio hop is indistinguishable from correct data at this layer.
- The sender must **not flood** the network or the receiver.

Every one of those requirements maps to a mechanism in this note. TCP is not a
protocol with reliability features; it is the accumulated answer to that list.

## The mental model: a byte stream, not packets

The single most useful reframe is this:

> TCP presents a **byte stream**. It does not present packets.

TCP numbers **bytes**, not messages. If you write 5000 bytes, TCP may segment
them however it likes — one 5000-byte burst, forty segments of 1250, or anything
in between — and the receiver's `read()` calls will return whatever chunks
happen to be available, which need not match your `write()` boundaries.

This has a practical consequence people trip over constantly: **you cannot use
TCP to find message boundaries.** A `read()` returning 100 bytes does not mean a
100-byte message arrived; it means 100 bytes were available. If your protocol
needs message framing, you must add it — a length prefix, a delimiter, or a
fixed-width field. HTTP/1.1 does this with `Content-Length` and chunked
encoding. This is the same class of problem as TCP's stream semantics vs UDP's
datagram semantics, and it is why "just use TCP" is not a complete protocol
design.

## Connection establishment: the three-way handshake

Before any data moves, the two endpoints exchange three segments to agree on
starting sequence numbers. Each side must learn the other's initial sequence
number and confirm its own was received.

### Step 1: client sends SYN

```text
  Sender → Receiver     Seq    Ack    Flags
1  Client → Server      1000   0      SYN
```

The client picks an **initial sequence number (ISN)** and sends it. The
acknowledgement field is meaningless in the first segment, so it carries nothing.

### Step 2: server sends SYN-ACK

```text
  Sender → Receiver     Seq    Ack    Flags
2  Server → Client      5000   1001   SYN, ACK
```

The server acknowledges the client's SYN by setting `Ack = 1001`. Read that
arithmetic carefully, because it is the whole trick: **the acknowledgement number
is always the next byte expected, so it is always one more than the sequence
number just received.** The client's SYN consumed sequence number 1000, so the
server expects 1001 next. The server also picks its own ISN, 5000.

### Step 3: client sends ACK

```text
  Sender → Receiver     Seq    Ack    Flags
3  Client → Server      1001   5001   ACK
```

The client confirms the server's SYN with `Ack = 5001`, using the same rule. Both
sides now know the other's starting point, and the connection is established.

```text
      Client                                Server
        |                                     |
        |---- seq=1000, ack=0, SYN --------->|   1
        |                                     |
        |<--- seq=5000, ack=1001, SYN,ACK ---|   2
        |                                     |
        |---- seq=1001, ack=5001, ACK ------>|   3
        |                                     |
        |==== connection established ========|
        |                                     |
        |==== application data flows =========|
```

Three segments, one round trip, and **no data moves until all three complete.**
This is a real cost, paid on every new connection, and it is the first thing
later protocols attack:

- HTTP keep-alive amortizes it across many requests on one connection.
- HTTP/2 multiplexes many requests over a single connection, paying the handshake
  once.
- HTTP/3's QUIC carries the transport handshake inside the TLS handshake, and
  supports 0-RTT resumption so a returning client can send data on the first
  flight.

If you watch a socket during a connection attempt, this handshake is exactly what
the `SYN-SENT` state means:

```bash
ss -tn state syn-sent
```

A `SYN-SENT` socket that persists is the handshake failing, not your application
hanging. That distinction saves a lot of debugging time.

### Why random initial sequence numbers?

ISNs must be unpredictable. Modern stacks randomize them, which is a security
requirement rather than an aesthetic one: if an attacker can guess the ISN, they
can inject forged segments into an established connection. This is why RFC 6528
exists ("TCP Initial Sequence Number (ISN) Generation") and why a stack with
sequential or time-based ISNs is a genuine vulnerability.

The example values 1000 and 5000 above are teaching values, not what a real
stack sends.

## The header

Every segment carries a header. It is **20 bytes minimum** and up to 60 bytes
when options are present. Field layout per RFC 9293 §3.1, verified against the
specification:

```text
  bit  0                   1                   2                   3
      0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
     +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
     |          Source Port          |       Destination Port        |
     +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
     |                         Sequence Number                        |
     +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
     |                    Acknowledgment Number                      |
     +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
     | Data Offset |Rsv|N|C|E|U|A|P|R|S|F|             Window          |
     +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
     |            Checksum             |          Urgent Pointer      |
     +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
     |                   Options (0 to 40 bytes)                      |
     +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
     |                        Data / Payload                          |
     +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

  Reading the Data Offset + flags row, left to right:

    Data Offset  4 bits   header length in 32-bit words. 5 = 20 bytes, 15 = 60.
    Rsv          3 bits   reserved, must be zero
    N            1 bit    NS — nonce protection against wrapped sequence numbers
    C            1 bit    CWR — congestion window reduced
    E            1 bit    ECE — ECN echo
    U            1 bit    URG — urgent pointer field is significant
    A            1 bit    ACK — acknowledgment number is significant
    P            1 bit    PSH — deliver to the application promptly
    R            1 bit    RST — reset the connection
    S            1 bit    SYN — synchronize sequence numbers
    F            1 bit    FIN — no more data from sender
    Window       16 bits  receive buffer space, in bytes

  Eight flags (C E U A P R S F) plus NS. A receiver ignores any flag bit it does
  not understand, which is how flags were added over decades without breaking
  old peers.
```

The header is a fixed grid of 32-bit rows, and reading it on the wire means
counting bits. Two consequences follow directly:

**Minimum size is a constraint of the layout, not a preference.** With no options
the header is 5 rows of 4 bytes = 20 bytes. Adding options extends to at most
15 rows = 60 bytes. The Data Offset field is what tells the receiver where the
payload starts, which is why a receiver must read it before anything else.

**Most fields sit on their own 32-bit boundary for alignment.** Sequence and
acknowledgement numbers, checksum, and the window are each a full row. This
matters if you are parsing packets on an architecture where unaligned reads are
slow or fault — the layout is deliberately friendly to that.

Fields that matter most, in order of how often they explain a problem:

| Field | Bytes | What it does |
|---|---|---|
| Source / Destination port | 2 + 2 | Which process on each host |
| Sequence number | 4 | Byte offset of this segment's first data byte |
| Acknowledgement number | 4 | Next byte the receiver expects |
| Data offset | 4 bits | Header length, 5–15 words of 4 bytes |
| Reserved + flags | 1 byte | SYN, ACK, FIN, RST, PSH, URG, ECE, CWR, NS |
| Window | 2 | How many more bytes the receiver can accept |
| Checksum | 2 | Detects corruption of header **and** data |
| Urgent pointer | 2 | Offset to urgent data; see below |
| Options | 0–40 | MSS, window scale, timestamps, SACK |

Fixed rows sum to 20 bytes: 4 (ports) + 4 (sequence) + 4 (acknowledgement) +
2 (offset + flags + window) + 2 (checksum) + 2 (urgent pointer). Options and
payload follow.

### Ports and demultiplexing

Ports are how TCP knows which process on a host should receive the data. The
incoming segment arrives at a host, not to an application; the destination port
selects the receiving socket.

This is the distinction worth internalizing: **an IP address gets the packet to a
host; a port gets it to a process.** A server listening on `0.0.0.0:443` accepts
on any local address; `127.0.0.1:443` accepts only loopback. If a connection
refuses on one interface and works on another, this is why.

### Sequence and acknowledgement numbers

These are byte counters, and off-by-one confusion here causes real bugs.

- **Sequence number**: the byte offset of the first byte in this segment,
  relative to the start of the stream.
- **Acknowledgement number**: the next byte the sender should transmit. It is
  *cumulative* — acknowledging 1460 bytes acknowledges everything up to that
  point, which is exactly why reordering is handled so cheaply.

So after the first data segment carries bytes 1001–2460, the receiver's ACK is
2461. If it later receives 2461–3920, the next ACK is 3921. The receiver never
has to acknowledge each segment separately; one number covers everything before
it.

### Window: the receiver's say

The 16-bit window field advertises how many more bytes the receiver has buffer
space for. This is **flow control**: it protects the receiver.

```text
  Sender                                   Receiver
    |                                        |
    |== 1460 bytes of data ================>|   seq 1001
    |                                        |
    |                                        | buffer fills up
    |<====== ack=2461, window=2920 =========|   "I have room for ~2 more"
    |                                        |
    |== 1460 bytes of data ================>|   seq 2461
    |                                        |
    |<====== ack=3921, window=0 =========== |   "stop, I am full"
    |                                        |
    |   sender stops sending                |
    |                                        | application drains buffer
    |<====== ack=3921, window=1460 ======== |   "room again"
    |                                        |
    |== resumes sending ===================>|
```

Note the 16-bit field maxes out at 65535 bytes, which is far too small a window
for any real network. The **window scale option** (RFC 7323) multiplies the
advertised window by a power of two, chosen during the handshake, taking
effective windows into the gigabyte range. Without it, TCP cannot fill a fast
link on a long path — the sender would stall waiting for window updates every
64 KB. If you see `tcp_window_scaling` mentioned in a tuning guide, this is why.

### Checksum

The checksum covers the header and the data, and the receiver verifies it. A
mismatch means corruption, and the segment is **discarded silently** — the sender
learns about it only by the absence of an acknowledgement.

This is a deliberately quiet failure, and it interacts badly with anything that
filters aggressively. A middlebox that corrupts a segment, or a broken path, will
produce retransmissions rather than an error message, because the sender cannot
distinguish "corrupted in transit" from "lost".

### Urgent pointer

The urgent pointer field, when the URG flag is set, carries an offset from the
sequence number to the byte marked as urgent. The intent was to let a sender
signal "this byte matters, interrupt the application".

In practice it is close to vestigial and the implementations do not agree. Linux
defaults to the BSD interpretation (the pointer addresses the byte *after* the
urgent data) because it interoperates with other stacks; RFC 1122 specifies the
opposite. The `tcp_stdurg` sysctl switches to the RFC interpretation, and the
`tcp(7)` man page notes the default "violates RFC 1122, but is required for
interoperability with other stacks."

Rather than use it, applications send out-of-band data — which arrives on the
same stream — or use a protocol-level priority mechanism. This is a good example
of a feature that outlived its usefulness and stayed in the protocol for
compatibility reasons.

## Data transfer and segmentation

TCP does not send whatever you hand it in one piece. It breaks the stream into
**segments** sized to fit the path's MTU.

The **maximum segment size (MSS)** is the largest payload a segment can carry
without IP fragmentation. On a 1500-byte MTU link with 20-byte IP and 20-byte TCP
headers:

```text
  1500  MTU
   -20  IPv4 header
   -20  TCP header
  ----
  1460  MSS
```

MSS is announced in a TCP **option** during the handshake, and each side learns
the other's. The value that matters is **the smaller of the two announced MSS
values** — you cannot send more than the receiver said it can accept, and you
cannot send more than your own path allows.

A subtlety worth knowing: **the MSS option is sent only in SYN segments.** There
is no MSS on an established connection's data segments, and no MSS in the `A`
record of a DNS response. If you are looking for an MSS and it is not in the
handshake, it is not on the wire. Note also that MSS is a *local* maximum for
what you may send; it is subtracted from the MTU, which is why it excludes the
IP and TCP headers.

A worked example, because arithmetic in a protocol note should be reproducible.
Transferring 3 MiB of data at MSS 1460:

```text
  total payload      3,145,728 bytes   (3 MiB, i.e. 3 x 1024 x 1024)
  MSS                          1,460 bytes
  3,145,728 / 1,460          = 2,154 remainder 888
  segments required              2,155   (last one carries 888 bytes)
```

Note the unit: this is 3 **MiB** (3 × 1024 × 1024 = 3,145,728), which is what a
program sees as "3 MB" of memory. A file of 3,000,000 bytes would need a
different count, which is why the worked example states the unit explicitly.

The source PDF for this module states 2155 segments, which is correct — but it
arrives there by rounding a division that has a remainder of 888 bytes. The
remainder is the interesting part: **the last segment is partial.** A note that
reports only the count hides the fact that not every segment is full, which
matters when you are reasoning about packet counts, ACKs, or buffer sizing.

### The sliding window

TCP does not wait for every segment to be acknowledged. It sends a window of
unacknowledged data, and slides that window forward as acknowledgements arrive.
The window size is bounded by both flow control (the receiver's advertised
window) and congestion control (what the network will bear).

```text
  Sent and ACKed | Sent, awaiting ACK | Not yet sent
  <--------------|-------------------|------------->
                 ^                    ^
                 window left edge     right edge

  As ACKs arrive from the left, the window slides right.
  Sender may transmit whenever bytes are in the "not yet sent" region
  and both limits allow.
```

The two limits are genuinely different and get confused constantly:

- **Flow control** protects the *receiver*. It is governed by the advertised
  window and is about the receiver's buffer capacity.
- **Congestion control** protects the *network*. It is governed by TCP's own
  estimate of network conditions and is about not creating packet loss for
  everyone else.

A transfer can be limited by either, and the symptoms differ: flow-control-limited
means the receiver is the bottleneck, congestion-limited means the path is.

### Nagle's algorithm: the small-packet problem

Sliding the window solves throughput for bulk transfers, but it leaves a problem
for request-response traffic. If your application writes 20 bytes, TCP has a
perfectly good window and will send it immediately — one 20-byte segment for 20
bytes of work. Do that a thousand times and you have spent a thousand round trips
and several thousand headers on a thousand tiny requests.

**Nagle's algorithm** reduces exactly this. Its rule: if there is unacknowledged
data in flight, a new small write is **buffered** until either the outstanding
data is acknowledged or enough accumulates to fill a segment.

```text
  write(20) ──► send immediately              (nothing outstanding)

  write(20) ──► hold. 20 bytes already unacked
  write(20) ──► hold. coalesce
  write(20) ──► hold. coalesce
                 │
                 ├── ack arrives ──► flush 60 bytes as one segment
                 └── a full MSS fills ──► send what we have
```

This converts three small writes into one segment, saving a round trip and two
headers. It is a genuine optimisation and it is enabled by default.

It is also a source of latency that surprises people badly. The cost is a
delayed small write:

- If you write 20 bytes and the previous data is unacknowledged, your write sits
  in the kernel until an ACK arrives. On a 200 ms path that is up to 200 ms of
  added latency for no benefit at all.
- In request-response protocols, this is exactly wrong: the client cannot send the
  next request until it has the response.

This is why `TCP_NODELAY` exists. Setting it disables Nagle, trading efficiency
for latency:

```go
conn.SetNoDelay(true)   // Go
```

Go sets `TCP_NODELAY` **on by default** for TCP connections, which is why Go
programs do not usually exhibit Nagle latency and C programs frequently do.
Node's `net` module also enables `noDelay` by default. Most other ecosystems
leave Nagle on.

The trade-off is real and neither setting is universally right. If you are
sending many small messages with no latency requirement, leave Nagle on. If you
have a request-response protocol on a high-latency path, turn it off.

## Acknowledgement and retransmission

TCP's loss detection is indirect. **There is no negative acknowledgement.** The
sender never learns "a packet was lost"; it learns only that an acknowledgement
did not arrive in time. Everything follows from that.

### The retransmission timer

Each unacknowledged segment is covered by a **retransmission timeout (RTO)**. If
the timer expires, the sender retransmits and **doubles** the RTO — exponential
backoff.

```text
  Segment sent ──── wait RTO ──── no ACK ──── retransmit
                                            RTO x 2
  Segment sent ──── wait 2RTO ─── no ACK ──── retransmit
                                             RTO x 4
  ... 1s → 2s → 4s → 8s → 16s → ...
```

The backoff is not arbitrary politeness. If a link is genuinely congested,
retransmitting immediately and aggressively makes it worse, and in the limit
causes congestion collapse. Backing off spreads the retries.

### How RTO is computed

RTO is derived from measured round-trip time, not fixed. RFC 6298 defines the
current algorithm using two smoothed values:

```text
  SRTT   smoothed round-trip time      (long-run average)
  RTTVAR round-trip time variation     (how jittery the path is)

  RTO = SRTT + max(G, K * RTTVAR)      K = 4, G = clock granularity
```

With smoothing factors `alpha = 1/8` and `beta = 1/4`. Key behaviours:

- **Initial RTO is 1 second** if no RTT has been measured yet (RFC 6298 §2.1;
  this was lowered from 3 seconds).
- **RTO is never below 1 second** after computation (§2.4) — a floor that
  deliberately makes TCP conservative.
- If the SYN or SYN-ACK is lost, RTO reverts to 3 seconds before data
  transmission (§5.7).
- Backoff on repeated loss is mandatory: `RTO <- RTO * 2` (§5.5).
- RTT samples must not come from retransmitted segments (Karn's algorithm,
  RFC 6298 §3) — a late ACK is ambiguous between the original and the retry, so
  sampling it would corrupt the estimate. The TCP timestamp option removes the
  ambiguity and is why timestamps are enabled by default.

So a connection to a 40 ms host might settle at an RTO around 1 second — the
floor, not the RTT. **This is the single most common source of confusion about
TCP timeouts**: the retransmission timer is not an estimate of your latency, it
is a conservative multiple of it with a hard 1-second floor.

Note that the 1-second floor is a *specification* requirement, not an
implementation artefact — some stacks have a lower internal floor, but a
conforming implementation rounds up to 1 second.

### How long until the connection is declared dead

RTO backoff alone would take an absurd time: doubling from 1 second to reach
20 minutes of total waiting is mathematically absurd to compute by hand, but the
point is that a conforming implementation should give up far sooner.

Real stacks therefore cap the attempt count. On Linux:

```bash
sysctl net.ipv4.tcp_retries2 net.ipv4.tcp_syn_retries
```

```text
net.ipv4.tcp_retries2 = 15
net.ipv4.tcp_syn_retries = 6
```

- **`tcp_retries2`** (default 15) caps retransmissions of an **established**
  connection. The `tcp(7)` man page states this corresponds to roughly
  **13 to 30 minutes** depending on the RTO. It also notes that the RFC 1122
  minimum of 100 seconds is "typically deemed too short" in practice.
- **`tcp_syn_retries`** (default 6) caps SYN retransmissions for a new
  connection, corresponding to about **127 seconds**.

The SYN figure is the one that bites. **A connection attempt to an unreachable
host fails after roughly two minutes**, not the "30 seconds" many people
remember from older systems — the default was 5 retries (about 180 seconds)
before Linux 3.7. This is why `connect()` timeouts feel so long, and why
application-level timeouts are usually set well below this.

### Retransmission is expensive

A lost segment costs you the full RTO before the retransmission goes out. On a
50 ms path with a 1-second RTO floor, losing one segment stalls the stream for
about a second — twenty round trips' worth of idle time. This is why packet loss
hurts interactive TCP so much more than its bandwidth cost suggests, and it is
the core motivation for QUIC's more aggressive retransmission.

Linux mitigates this where it can. `tcp_frto` (enabled by default) handles
**spurious timeouts** — cases where the RTO expired but the ACK was merely
delayed rather than lost, which is common on wireless links where radio
interference causes random loss rather than congestion. Without it, TCP would
treat a false alarm as real congestion and needlessly halve its congestion
window.

### Duplicate ACKs and fast retransmit

Waiting a full RTO is slow when the loss is detectable sooner. If a sender
receives **three duplicate ACKs** — the same acknowledgement number three times
— it infers a segment was lost rather than merely reordered, and retransmits
immediately without waiting for the RTO.

```text
  seq 1001 ──> lost
  seq 2461 ──>            ack 1461
  seq 3921 ──>            ack 1461   (duplicate 1: something is missing)
  seq 5381 ──>            ack 1461   (duplicate 2)
  seq 6841 ──>            ack 1461   (duplicate 3) → fast retransmit seq 1001
```

The reasoning: ACKs are cumulative, so receiving an ACK for 6841's predecessor
while data is missing means 1461 is the hole. Three is chosen because
reordering is common and two duplicates can be caused by it; three makes a false
positive unlikely.

**Selective acknowledgement (SACK)** improves this further. Rather than only
saying "I got up to 1461", a receiver with the SACK option can say "I got up to
1461, and I also got 2461–3920", letting the sender retransmit only the actual
hole instead of everything from the gap onward.

## Connection teardown

Closing is asymmetric and takes four segments, because each direction must be
drained independently.

```text
      Client                                Server
        |                                     |
        |---- FIN, seq=u ------------------->|   no more data from client
        |<--- ACK, ack=u+1 -------------------|   server acknowledges
        |                                     |   (server may still send)
        |<--- FIN, seq=v --------------------|   server's turn to close
        |---- ACK, ack=v+1 ------------------>|   connection closed
```

The middle gap matters: after the client sends FIN it can still **receive** data.
A half-closed connection is a normal state, and a protocol that assumes
close means "nothing more from either side" will lose data.

**RST** is the abrupt version — connection reset, no negotiation. You see it
when you connect to a port with nothing listening, or when a middlebox has killed
the connection. If an application sees `ECONNRESET` where it expected clean
shutdown, suspect something in between rather than the application.

## Connection states

TCP sockets move through states, and `ss` shows them. The full set is large, but
these are the ones that explain behaviour you will actually observe:

| State | Meaning |
|---|---|
| `LISTEN` | Server socket waiting to accept |
| `SYN-SENT` | Client sent SYN, waiting for SYN-ACK |
| `SYN-RECV` | Server sent SYN-ACK, waiting for final ACK |
| `ESTAB` | Handshake complete, data can flow |
| `FIN-WAIT-1` / `FIN-WAIT-2` | This side closed, awaiting peer |
| `CLOSE-WAIT` | Peer closed; **this side must still close** |
| `LAST-ACK` | Waiting for final ACK after local close |
| `TIME-WAIT` | Waiting 2×MSL before the connection can be discarded |
| `CLOSE` | Closed |

**`CLOSE-WAIT` is the one that causes leaks.** It means the peer sent FIN and you
have not closed your end. If your application closes its socket and then leaks
the file descriptor on every response, you accumulate `CLOSE-WAIT` sockets until
you exhaust descriptors. This is one of the most common real TCP bugs in
long-running servers.

**`TIME-WAIT` exists to protect the network.** It lingers for twice the maximum
segment lifetime so that delayed duplicates from an old connection cannot be
mistaken for data on a new connection reusing the same 4-tuple. It looks like a
waste — a port held for minutes after closing — and it is a deliberate
correctness trade-off. Servers that appear to run out of ephemeral ports are
usually hitting `TIME-WAIT` accumulation.

## Practical experiment: watch it happen

The handshake is visible with ordinary tools. Watch the socket state while a
connection is being made to a host that will not answer, so the handshake hangs
in a state you can read:

```bash
# In one terminal
ss -tn state syn-sent

# In another, connect to a non-responsive address with a long timeout
nc -v 203.0.113.1 443
```

The socket appears in `SYN-SENT` and stays there. That is the SYN going out and
no SYN-ACK returning — you are watching the first step of the handshake stall,
and the eventual timeout is RTO backoff in action.

To see completed handshakes and their timing:

```bash
curl -o /dev/null -s -w 'dns=%{time_namelookup}s connect=%{time_connect}s tls=%{time_appconnect}s total=%{time_total}s\n' https://example.com
```

`connect` is the TCP handshake, isolated from DNS and TLS. Compare it against
`total` — the handshake is usually a small fraction of a page load, which is the
quantitative form of the "keep-alive amortizes it" point above.

## Common misconceptions

**"TCP guarantees delivery, so I don't need to worry about it."**
TCP guarantees delivery of the byte stream *or* it reports a failure. It does not
guarantee that a write succeeded: the data may be sitting in a kernel buffer,
unacknowledged, when your process is killed. Durable delivery needs
`fsync`, a database, or an application-level acknowledgement. This is the single
most important misconception on this page — it is why "the client said it failed
but the data is in the database" happens.

**"A segment is a message."**
Segments are an implementation detail of the transport. Your writes and the
receiver's reads do not align with them, and you cannot rely on segment
boundaries to find message boundaries. Add explicit framing.

**"The RTO is my latency."**
No. RTO is a smoothed multiple of measured RTT with a 1-second floor, doubled on
each failed retransmission. On a fast path the floor dominates entirely.

**"TCP's checksum means data is not corrupted."**
It means corruption is *detected*. A mismatched segment is discarded silently and
retransmitted. End-to-end integrity still depends on your application checking
what it receives.

**"UDP is faster because it skips steps TCP does."**
UDP skips the handshake, acknowledgements, retransmission, and flow control. It
does not skip the network, and it does not go faster on a path with no loss.
Where loss exists, UDP hands that problem to the application, which usually
solves it worse than TCP did.

**"TCP is the internet."**
TCP is one transport. QUIC over UDP carries HTTP/3 and most new traffic. The
internet is IP; TCP is one thing that runs on it.

## Trade-offs

**Reliability costs latency and complexity.** A lost segment stalls a stream for
up to an RTO. Head-of-line blocking means one lost segment prevents delivery of
everything behind it, even data that arrived intact. This is why real-time
applications often accept loss instead.

**Every endpoint pays.** Because the network core is deliberately dumb, every TCP
implementation reimplements congestion control, retransmission, and ordering.
That is the price of the end-to-end principle, and it is why a new transport
means reimplementing all of it.

**Connection setup costs a round trip.** Paid per connection unless amortized by
multiplexing or resumption. For workloads dominated by small requests to many
hosts, this is a large fraction of total time.

**State is per-connection.** Enough state to track buffers, sequence numbers, and
timers means a server pays memory and scheduler cost per connection, which is why
connection counts are an operating metric.

**The network makes no guarantees, so TCP must.** There is no QoS to reserve
capacity. TCP's response is to be conservative and back off, which protects the
internet but means a congested path degrades for everyone rather than degrading
for the traffic that deserves priority.

## Related concepts

- [All about the internet](./01-all-about-the-internet.md) — the packet-switching
  model and the end-to-end principle this note depends on.
- [How data transfers](./02-how-data-transfers.md) — encapsulation, ARP, and how
  a segment actually gets onto a wire.
- [UDP](./05-udp.md) — the same transport layer without any of this machinery.
- [DNS](./03-dns.md) — runs on top of this, and is on the critical path of every
  connection.
- Connection pooling and keep-alive — where the handshake cost is amortized.
- QUIC and HTTP/3 — where this note's costs are being attacked.

## References

- RFC 9293 — Transmission Control Protocol (STD 7). The current TCP
  specification. https://www.rfc-editor.org/rfc/rfc9293
- RFC 6298 — Computing TCP's Retransmission Timer. The RTO algorithm described
  above, including the 1-second initial value and the 3-second SYN-loss revert.
  https://www.rfc-editor.org/rfc/rfc6298
- RFC 5681 — TCP Congestion Control. The congestion control requirements.
  https://www.rfc-editor.org/rfc/rfc5681
- RFC 5682 — TCP Congestion Window. How the window is determined.
  https://www.rfc-editor.org/rfc/rfc5682
- RFC 7323 — TCP Window Scale Option. Why the 16-bit window needs scaling.
  https://www.rfc-editor.org/rfc/rfc7323
- RFC 6528 — TCP Initial Sequence Number (ISN) Generation. Why ISNs are
  randomized. https://www.rfc-editor.org/rfc/rfc6528
- RFC 2018 — TCP Selective Acknowledgment Options.
  https://www.rfc-editor.org/rfc/rfc2018
- RFC 793 — Transmission Control Protocol. The original specification, now
  obsoleted by RFC 9293. https://www.rfc-editor.org/rfc/rfc793
- A. Tanenbaum, D. Wetherall, *Computer Networks*, 6th ed., Pearson, 2022,
  ch. 6. Useful background on transport-layer reliability; the header layout
  above was drawn from RFC 9293 rather than reproduced from any text.
- J. Postel, "Transmission Control Protocol", RFC 793, 1981.
- Linux man-pages: `tcp(7)`, `ss(8)`, `net.ipv4.tcp_retries2(5)`.
  https://man7.org/linux/man-pages/man7/tcp.7.html
