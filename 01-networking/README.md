# 01-networking

What the internet is, how bytes actually move across it, and the protocols
that make that movement possible.

This is the foundational module. Almost everything downstream — HTTP, databases,
caching, service-to-service traffic — is carried by machinery explained here. If
a later module assumes you understand routing, latency, or a byte stream, this is
where that comes from.

## Scope

**In this module**

- What the internet is, structurally, and why it has no owner.
- Packet switching versus circuit switching, and why packet switching won.
- Encapsulation, ARP, and how a packet is addressed differently at each layer.
- The transport layer: TCP's reliability machinery and UDP's deliberate lack of
  it.
- Name resolution with DNS.
- How to observe all of the above on your own machine.

**Deliberately elsewhere**

- HTTP semantics and the request/response model → module 02.
- TLS and encryption at the transport edge → module 02.
- Application-level protocols (gRPC, message queues) → later modules.
- Detailed congestion control math (slow start, CUBIC, BBR) → beyond these notes.
- Anything OS-specific (system calls, sockets, buffering) beyond what is needed
  to explain the network itself → module 03.

## Reading order

| # | Note | Prerequisites | Status |
|---|---|---|---|
| 01 | [All about the internet](./01-all-about-the-internet.md) | none | stable |
| 02 | [How data transfers](./02-how-data-transfers.md) | 01 | draft |
| 03 | [DNS](./03-dns.md) | 01, 02 | draft |
| 04 | [TCP](./04-tcp.md) | 01, 02 | draft |
| 05 | [UDP](./05-udp.md) | 01, 02 | draft |

DNS sits before TCP deliberately. TCP's mechanisms are easier to understand once
you have seen a UDP exchange in the wild, and note 03 contains real `dig` output
you can reproduce.

## Start here

Read [All about the internet](./01-all-about-the-internet.md) first, in full. It
is the orientation note: it establishes the vocabulary and the mental model every
other note here depends on, and it contains the first practical experiment
(`ip`, `ping`, `tracepath`) that teaches you to see the network rather than infer
it.

Then [How data transfers](./02-how-data-transfers.md), which answers the question
that trips up most people: the IP address never changes, but the MAC address is
rewritten at every single hop.

## Examples

Runnable reference implementations, each a single file with no dependencies.

| Example | Demonstrates |
|---|---|
| [build-tcp-server](./examples/build-tcp-server/) | The handshake is kernel work, and TCP is a byte stream — in Go and TypeScript |
| [build-udp-server](./examples/build-udp-server/) | No handshake, message boundaries preserved, silent loss |

They are written to be **copied into your own working environment**, not built as
a project. Each has its own README covering prerequisites, how to run it, what to
watch for, and what to change to learn more.

## Experiments

Every note in this module contains commands that were actually run, with real
output. Notes 01 and 02 cover the commands you will use most:

```bash
ip -brief addr          # interfaces and addresses
ip route                # routing table
ip route get <ip>       # what the kernel will do, without sending
ip neigh show           # the ARP cache
ping -c 3 <ip>          # round-trip time and loss
tracepath -n <ip>       # the hop path, and MTU along it
ss -tnp                 # TCP sockets and their states
ss -uapn                # UDP sockets, note UNCONN
dig +trace <name>       # DNS resolution from the root down
```

## Diagrams

All diagrams in this module are ASCII, in ` ```text ` fences. Wire-format
headers — the TCP and UDP header layouts in notes 04 and 05 — are drawn as bit
grids rather than images, with field sizes verified against RFC 9293 §3.1 and
RFC 768 respectively. They diff as text, render identically on GitHub and in a
terminal, and stay correct when a field size is corrected.

## Source material

This module was written with reference to PDFs kept outside the repository in
`pdfs/` (gitignored). Those documents were treated as a starting point rather
than an authority — several claims did not survive checking against the
specifications. Notable corrections, documented in the notes themselves:

- DNS is **not** "managed by ICANN". ICANN coordinates the root zone and delegates
  TLDs; it operates one of the thirteen root server identities and runs no
  resolvers. See [DNS](./03-dns.md).
- `3145728 ÷ 1460` yields 2154 segments plus a remainder of 888 bytes, so 2155
  segments with the last one partial. See [TCP](./04-tcp.md).
- The TCP retransmission timeout has a **1-second floor** per RFC 6298 §2.4, with
  a revert to 3 seconds if the SYN is lost — not the "200ms to 3 seconds" range
  the source stated. See [TCP](./04-tcp.md).
- UDP's unit is a **datagram**, not a "segment". See [UDP](./05-udp.md).

## Status

**In progress.** 5 notes written — 1 stable, 4 draft. 2 example projects,
verified and runnable.
