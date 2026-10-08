---
title: UDP
module: 01-networking
status: draft
prerequisites:
  - ./01-all-about-the-internet.md
  - ./02-how-data-transfers.md
---

# UDP

UDP sends a message as a single datagram with no setup, no acknowledgement, no
retransmission, no ordering, and no flow control. It adds 8 bytes of header to
the payload and hands it to the network to deliver or lose.

That is the entire specification, and the honest question is why such a limited
thing is so useful. The answer is that UDP is not a worse TCP — it is a
deliberate refusal to implement four features at the transport layer, and for a
large class of traffic those features are not just unnecessary but harmful.

## Why does this exist?

TCP solves reliability by making assumptions that only the endpoints can verify:
that data loss is always a mistake to be corrected, that every byte must arrive,
that delay is worse than loss, and that one shared connection's worth of
throughput is what the application wants.

Each assumption is sometimes wrong. UDP exists so that an application can opt out
of them individually and implement only what it actually needs.

**Some data is worthless if late.** A video frame that arrives 200 ms after its
deadline is not degraded, it is discarded. Retransmitting it wastes bandwidth and
makes the video *later*. For live video, voice, and games, loss is tolerated and
latency is fatal — the opposite of TCP's priorities.

**Some loss is cheaper than retransmission.** A single dropped packet in a
1,000-packet video frame costs one frame; a retransmission costs an RTO of stall.
Real-time transport protocols routinely accept a 1–5% loss rate rather than
transmit redundancy.

**The application knows things the transport does not.** Whether a loss matters
depends entirely on the payload. A dropped byte in a video frame is invisible; a
dropped byte in a bank transfer is a corruption. TCP must pick one policy for all
traffic, because it does not know what it is carrying. An application can decide
per message — this is the strongest argument for the protocol existing at all.

**Setup cost is unacceptable for one-shot messages.** A DNS query, a service
discovery ping, a telemetry sample: three round trips and retransmission
machinery to deliver one small datagram that may not even need a reply. UDP sends
it in one.

## The mental model: messages, not bytes

The critical contrast with TCP:

> UDP preserves **message boundaries**. TCP does not.

One `send()` on a UDP socket produces exactly one datagram, and the receiver's
`recv()` returns that whole datagram or nothing. If you send three 100-byte
messages, the receiver gets three 100-byte messages — in whatever order, with
gaps possible, but never merged and never split.

This is a genuinely different abstraction, not a weaker version of the same one.
TCP is a reliable byte pipe; UDP is an unreliable mailbox with each item
enveloped separately.

```text
  TCP                                 UDP
  ----                                 ---
  send(100) send(100) send(100)        send(100) send(100) send(100)
        |        |        |                  |        |        |
        +--------+--------+                  |        |        |
        |  one continuous     |              v        v        v
        |  byte stream,       |            recv()   recv()   recv()
        |  boundaries lost    |            100      100      100
        v                                each exactly as sent
  recv(250) recv(50) ...
  no relation to your sends
```

The practical implication: **if your application needs message boundaries, UDP
gives them to you for free and TCP does not.** Conversely, if you need reliable
ordered delivery, UDP gives you nothing and TCP does.

## The header

Eight bytes, fixed, and that is the whole header. Layout per RFC 768:

```text
  byte    0           1           2           3
          +-----------+-----------+-----------+-----------+
  bit     0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
          +-----------+-----------+-----------+-----------+
          |                Source Port                |
          +-----------+-----------+-----------+-----------+
          |             Destination Port             |
          +-----------+-----------+-----------+-----------+
          |                   Length                  |
          +-----------+-----------+-----------+-----------+
          |                  Checksum                 |
          +-----------+-----------+-----------+-----------+
          |                  Data ...                 |
          +-----------+-----------+-----------+-----------+

  Four fields, each exactly 16 bits, two rows of 32 bits total. There is no
  version, no flags, no options, and no field that can grow. The Length field
  counts header plus payload, so a receiver knows how many bytes to expect
  before reading any of them.
```

Four fixed fields is the entire design. Compare it to TCP's ten, several of which
are variable-length.

| Field | Bytes | Meaning |
|---|---|---|
| Source port | 2 | Optional. `0` if unused |
| Destination port | 2 | Which process on the destination host |
| Length | 2 | Header + payload, so the receiver can size its buffer |
| Checksum | 2 | Corruption detection over header + payload |

Compare that to TCP's 20-byte minimum and its optional 40 more. **UDP's header is
40% the size of TCP's minimum** — 8 against 20 — and the gap widens once TCP's
options are present, which they nearly always are in practice.

That matters when you are sending many small messages, because header overhead is
a fixed cost per message:

| Payload | TCP overhead | UDP overhead |
|---|---|---|
| 100 bytes | 20/120 = 16.7% | 8/108 = 7.4% |
| 1,000 bytes | 20/1020 = 2.0% | 8/1008 = 0.8% |
| 1,400 bytes | 20/1420 = 1.4% | 8/1408 = 0.6% |

At 100-byte payloads, TCP spends roughly 17% of every message on its header and
UDP spends roughly 7%. Both figures collapse as messages get larger, so header
size is a small-message problem specifically — a protocol sending 4 KB chunks
should care about something else.

### The source port is optional

TCP needs a source port because ACKs must go back to a specific endpoint. UDP has
no acknowledgements, so it has no need for one, and you may send from port `0`.

This is why an outgoing UDP packet can be hard to attribute: a packet from
`1.2.3.4:0` to `5.6.7.8:53` is a legitimate DNS query with nothing in it
identifying the process that sent it. Application protocols fill in the source
port when they need a reply — DNS queries typically pick a random ephemeral port
so replies can be matched to the query at the client.

### The checksum is mandatory in IPv6

This trips people up. In IPv4 the UDP checksum may be zero, meaning "not
computed." **In IPv6 the checksum is required**, with zero reserved to mean "no
UDP packet at all," so it must never appear in a valid IPv6 datagram.

The reason is a real weakness rather than pedantry: a zero checksum in IPv4 means
corruption is not detected, and because UDP has no other mechanism, a corrupted
payload is silently delivered to the application as if it were valid. For a
protocol carrying DNS responses or media, that is unacceptable, so IPv6 removed
the option.

Note that the checksum is only **error detection**. It cannot correct anything —
the receiver has no way to know what the original bytes were. A checksum failure
means the datagram is discarded, and with no retransmission mechanism, that data
is simply gone.

## Why DNS runs on UDP

DNS queries are small, need a fast single response, and are valuable enough that
losing one is not fatal — the client just asks again. All of UDP's apparent
weaknesses are advantages here.

But DNS is also the instructive exception: when a response is too large for a UDP
datagram, DNS falls back to **TCP**. Zone transfers are always TCP, and
DNSSEC-signed responses are large enough that they require EDNS0 or TCP
routinely. This is a useful reminder that "UDP is unreliable" does not mean "UDP
is always used" — it means the choice is made per message, and the fallback
exists.

## Choosing between TCP and UDP

The honest version of this comparison:

| | TCP | UDP |
|---|---|---|
| Header overhead | 20–60 bytes | 8 bytes |
| Setup | 1 round trip (handshake) | None |
| Delivery | Retransmitted until acknowledged | Best effort, once |
| Ordering | Guaranteed | Not guaranteed |
| Duplicates | Never delivered to the app | Possible |
| Message boundaries | Lost | Preserved |
| Flow control | Receiver-advertised window | None |
| Congestion control | Yes | **None — your problem** |
| Payload limit | Stream, no inherent limit | 65,507 bytes max |
| Latency under loss | Stalls for an RTO | No stall |

The row that decides most real decisions is **congestion control**, and it deserves
more attention than it usually gets.

### Uncontrolled UDP can break the internet for everyone

TCP slows down when it encounters loss, because it interprets loss as a signal
that the path is congested. UDP does not, because it has no mechanism to do so.

So a UDP flood does not just consume its own share of bandwidth — it generates
loss on shared links, and **every TCP connection crossing those links slows
down**, since their congestion control sees the loss as congestion. You degrade
the network for other people's traffic, not just your own.

This is not hypothetical and not new. DDoS amplification attacks exploit it
directly: a small forged request to an open DNS resolver produces a much larger
response directed at the victim's address, using bandwidth the attacker did not
pay for. The original amplification work (RFC 5357, 2008) described DNS
reflection as the most effective vector at the time.

For an application choosing UDP on a shared network, the practical consequence is
that **you are responsible for not overwhelming the path.** QUIC's flow control
exists largely to give modern UDP-based transports this capability back.

### When to choose which

Choose **TCP** when you need: correctness and completeness, ordering, request-
response where missing a response is bad, or simply want the problem handled.

Choose **UDP** when: you need message boundaries preserved, you are
loss-tolerant and latency-sensitive (live media, voice, games), you are sending
many small one-shot messages, you need multicast, or you intend to implement
reliability yourself and want control over how.

The most common real-world answer is neither. **QUIC** runs over UDP and
reimplements what TCP provides — reliability, ordering, congestion control — with
a faster handshake, per-stream multiplexing without head-of-line blocking, and
encryption integrated into the transport. It exists because UDP gets you the
header size and the message boundaries while letting you build the reliability
layer better than TCP's design allows. HTTP/3 uses it, and it carries the majority
of web traffic today.

## Practical experiment: watch UDP on the wire

See your own resolver's UDP sockets:

```bash
ss -uapn
```

```text
State  Recv-Q Send-Q     Local Address:Port    Peer Address:Port  Process
UNCONN 0      0           172.17.0.1:53        0.0.0.0:*          
UNCONN 0      0           127.0.0.54:53        0.0.0.0:*          
UNCONN 0      0           127.0.0.53%lo:53     0.0.0.0:*          
UNCONN 0      0           224.0.0.251:5353     0.0.0.0:*    users:(("brave",pid=2336,fd=321))
```

Real output. Note the state is **`UNCONN`, not `ESTAB`** — this is the visible
difference between the two protocols. A UDP socket is not "established" because
there is nothing to establish. It exists, it sends, and it may receive, but no
relationship was ever negotiated.

The `Send-Q` column at `0` means nothing is waiting to go out. Watch it while
generating traffic and you will see it move, which is how you tell a UDP send
that the network has not yet accepted from one that was silently dropped
earlier.

Send a DNS query explicitly over UDP, bypassing any local resolver:

```bash
dig @1.1.1.1 +noall +stats example.com
```

```text
;; Query time: 21 msec
;; SERVER: 1.1.1.1#53(1.1.1.1) (UDP)
```

The `(UDP)` suffix is `dig` telling you the transport it used, and it is the point
of the command — you have forced a UDP exchange with a specific server with no
local caching layer in between.

Compare with TCP to the same server, which forces the fallback path:

```bash
dig @1.1.1.1 +tcp +noall +stats example.com
```

```text
;; Query time: 24 msec
;; SERVER: 1.1.1.1#53(1.1.1.1) (TCP)
```

Same answer, more bytes, slightly slower. The difference is the cost of
reliability — which, for a small query whose response would have arrived fine
over UDP, is pure overhead. That is precisely why DNS prefers UDP and falls back
only when it must.

### What to try next

- Write a minimal UDP echo server and client in Go's `net` package using
  `ListenUDP` and `ReadFromUDP`. Note how little API there is — no handshake, no
  accept loop, no connection object. Compare that surface area to the TCP version.
- Send a datagram larger than 65,507 bytes and observe the error. Then find where
  that number comes from: a 65,535-byte IP payload minus a 20-byte IPv4 header
  minus the 8-byte UDP header.
- Send to a closed port on a host that responds with ICMP port unreachable, and
  see how the error arrives asynchronously to your send call — or does not arrive
  at all if a firewall drops it.
- Count how much of a small UDP payload survives to the wire. At 20 bytes of
  payload, the 8-byte UDP header plus a 20-byte IP header plus 14 bytes of
  Ethernet framing means the overhead is nearly as large as the data.

## Common misconceptions

**"UDP is unreliable, therefore it's bad."**
Unreliable is a property, not a defect. For live media, a late frame and a lost
frame are equally useless, so retransmission is pure waste. UDP is the right tool
for data where timeliness matters more than completeness.

**"UDP is faster than TCP."**
UDP costs less to *send*; it does not travel faster. Both use the same network,
the same paths, and the same links. On a lossy path, TCP can end up delivering
*more* data faster because it retransmits what UDP lost.

**"UDP is just TCP without the reliability."**
They are different abstractions, not one minus the other. TCP is a reliable byte
stream with boundaries lost; UDP is an unreliable message stream with boundaries
preserved. Neither is a subset of the other, and choosing between them means
deciding which of those two properties you need.

**"UDP has no error detection."**
It has a checksum covering header and payload. It cannot correct errors, so a
failed datagram is discarded rather than repaired — but corruption is still
detected, and in IPv6 the checksum is mandatory rather than optional.

**"TCP guarantees delivery, UDP doesn't, so I should use TCP."**
TCP guarantees the byte stream arrives, or the connection is reported as failed. It
does not guarantee your write was durable, and it cannot tell you whether the
application on the other end acted on what it received. For many workloads that
guarantee is not the one you need.

**"UDP is only for video and games."**
DNS, DHCP, NTP, SNMP, syslog, WireGuard, and QUIC all run over UDP. Most of the
internet's new protocols do, because the flexibility is worth more than the
built-in reliability.

## Trade-offs

**No congestion control, by default.** The most serious trade-off, and the one
with consequences beyond your own traffic. A UDP application that ignores
congestion can degrade the network for every TCP connection sharing the path.

**No recovery from loss.** Data is gone. Applications must either tolerate the gap
or implement recovery themselves — and most implement it worse than TCP did,
because they lack decades of tuning and network feedback.

**The 65,507-byte payload limit.** A consequence of the 16-bit length field and
the 16-bit total-length field in IPv4. It forces large transfers over TCP or into
IP fragmentation, which is itself a reliability hazard.

**Application-level work is unbudgeted.** Reliability, ordering, congestion
control, and flow control all become your code. QUIC is evidence of what that
costs: a large, carefully engineered protocol whose main achievement is
reimplementing what TCP already provided.

**No inherent message size limit, but a hard one anyway.** Because you cannot
exceed 65,507 bytes, and because huge datagrams fragment, application protocols
built on UDP routinely add their own framing — QUIC's packet and frame encoding is
a direct example.

**Connectionless means stateless, which means unauthenticated.** There is no
handshake to establish identity, so a UDP service is open to anyone who can send
to its port. Every secure UDP protocol must add authentication itself. This is
part of why UDP services are a disproportionate source of amplification attacks:
there is no handshake to require a victim to have participated in.

## Related concepts

- [All about the internet](./01-all-about-the-internet.md) — the best-effort
  network model that makes UDP's lack of guarantees normal rather than unusual.
- [How data transfers](./02-how-data-transfers.md) — how a datagram reaches the
  wire.
- [TCP](./04-tcp.md) — the full contrast: handshake, ordering, retransmission,
  flow and congestion control.
- [DNS](./03-dns.md) — the most important UDP application, and the clearest
  example of choosing UDP per message.
- QUIC and HTTP/3 — UDP plus reimplemented reliability, and why it now carries
  most web traffic.
- IP fragmentation — why large UDP datagrams are their own reliability problem.
- Amplification and reflection attacks — abuse of connectionless services.

## References

- RFC 768 — User Datagram Protocol. The entire specification, and famously
  short. https://www.rfc-editor.org/rfc/rfc768
- RFC 1122 — Requirements for Internet Hosts, §4.1.3.4. Host requirements for
  UDP, including the IPv4/IPv6 checksum rule difference.
  https://www.rfc-editor.org/rfc/rfc1122
- RFC 8200 — IPv6 Specification, §8.1. UDP checksum requirements in IPv6.
  https://www.rfc-editor.org/rfc/rfc8200
- RFC 9000 — QUIC: A UDP-Based Multiplexed and Secure Transport. What modern
  UDP-based reliability looks like. https://www.rfc-editor.org/rfc/rfc9000
- RFC 5357 — DNS Security Extensions. §4 discusses UDP-based amplification
  attacks against open resolvers. https://www.rfc-editor.org/rfc/rfc5357
- RFC 8085 — UDP Usage Guidelines. When UDP is and is not appropriate, and how to
  size datagrams. https://www.rfc-editor.org/rfc/rfc8085
- RFC 6928 — Increasing the UDP Maximum Transmission Unit. Why larger datagrams
  are hard to deploy safely. https://www.rfc-editor.org/rfc/rfc6928
- A. Tanenbaum, D. Wetherall, *Computer Networks*, 6th ed., Pearson, 2022,
  ch. 5. Useful background on transport protocols; the header layout above was
  drawn from RFC 768 rather than reproduced from any text.
- `udp(7)` — Linux man-pages. https://man7.org/linux/man-pages/man7/udp.7.html
