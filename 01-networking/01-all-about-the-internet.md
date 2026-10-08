---
title: All About the Internet
module: 01-networking
status: stable
prerequisites: []
---

# All About the Internet

The internet is not a network. It is roughly 75,000 independently administered
networks that have agreed to interconnect, plus the protocols and institutions
that let them route packets between each other without any of them being in
charge.

Nothing about that arrangement is guaranteed. There is no central router, no
central naming authority that resolves names to addresses, no central registry
of who owns what, and no operator who can shut it down. Every design decision in
this note exists because the original designers had to assume that other
administrations would be independent, unreliable, and possibly adversarial.

That single constraint — **no party is trusted, and no party is in charge** —
generates almost everything else: packet switching, the datagram model, the
end-to-end principle, the fact that IP addresses are not identities, and the
existence of NAT.

## Why does this exist?

Networks existed before the internet. In the 1960s a "network" meant a
proprietary system from one vendor, connecting one organization's terminals to
one organization's mainframe, or leased lines from a telephone company. Each
such network assumed a single owner, a known set of endpoints, and a central
point of administration.

That model broke down for three reasons.

**Interconnection.** Organizations needed to talk to each other, not just to
themselves. The early answer was the ARPANET (1969), which connected a few dozen
research institutions. It worked, but every participant had to be known and
approved in advance — adding a site required physical changes and central
coordination.

**Scale.** The number of sites grew by orders of magnitude. Any design that
required a central administrator to know about every endpoint, or to reconfigure
every node when one node was added, could not keep scaling.

**Failure.** Packet-switched networks were built from nodes that fail, links that
break, and traffic that arrives out of order. A network that assumes reliable,
ordered delivery across long distances is a network that stalls whenever anything
breaks.

The internet's design answers all three the same way: **push intelligence to the
edges, keep the middle as dumb as possible, and accept loss and disorder as
normal inputs rather than exceptions to handle.**

## What the internet is made of

### Autonomous systems

An **autonomous system (AS)** is a network under a single routing policy — one
organization's collection of routers that share a common routing table and
announce routes coherently to the rest of the world. ISPs, universities, large
corporations, and content providers all run autonomous systems. The count is in
the tens of thousands and grows every year.

The interior of one AS is private to that organization. The **Border Gateway
Protocol (BGP)** is the protocol that connects ASes to each other. BGP runs
between routers at the borders of networks, and it is the reason the internet has
no central structure: it lets any AS announce "I can reach these address ranges"
to its neighbors, and neighbors propagate that announcement onward.

BGP is a **path vector** protocol. Unlike an interior protocol, BGP advertises
the full sequence of autonomous systems a route traverses. This is deliberate
and has two consequences you will feel for the rest of your career:

- **Policy beats distance.** A BGP router can prefer a longer, more expensive
  path over a shorter one if the operator prefers it. Shortest-path routing, as
  used inside an AS, cannot express this. BGP is how commercial relationships and
  traffic engineering get expressed.
- **BGP is a trust relationship with no verification.** BGP authenticates that a
  peer *is* a peer. It does not verify that what a peer tells you is true. This
  is the root cause of route hijacking: if a network announces address space it
  does not own, some fraction of the internet will believe it. Mitigations
  exist — RPKI route origin validation, route collector data — but they are
  not universally deployed, so BGP remains the internet's largest
  unauthenticated surface.

```text
AS 64500 (Example Corp)         AS 64501 (Example ISP)
  ┌──────────────┐                ┌──────────────┐
  │ 10.0.0.0/16  │                │ 10.1.0.0/16  │
  │ private      │                │ private      │
  └──────┬───────┘                └──────┬───────┘
         │ BGP session                     │ BGP session
         │  announce 203.0.113.0/24        │  announce 198.51.100.0/24
         └───────────────┬─────────────────┘
                         │  eBGP: exchange routes
                 ┌───────┴────────┐
                 │ transit +      │
                 │ peering links  │
                 └────────────────┘

  203.0.113.0/24 and 198.51.100.0/24 are RFC 5737 documentation ranges.
  Real announcements carry the same structure, with routable addresses.
```

### Peering and transit

Autonomous systems interconnect in two ways.

**Transit** is paying a larger network to carry your traffic onward. It is the
default, and it costs money — often a significant amount of money.

**Peering** is two networks agreeing to exchange traffic directly, usually at an
**internet exchange point (IXP)** — a physical switch in a data center where many
networks meet. Peering is free, and both sides benefit because neither pays a
third party for the same traffic.

This economics explains a lot of otherwise-puzzling topology. A network carrying
heavy traffic between two regions will build direct links or buy peering to avoid
paying transit for traffic it both sends and receives. Content providers build
networks into regions and near IXPs precisely to move this balance. None of this
is visible in the addressing plan, and all of it shows up in your latency.

### The physical layer is not abstract

Everything above runs on physical media: fibre, copper, radio, and light through
space. A packet that crosses an ocean is an optical pulse in a fibre cable
underwater, not an abstract message. Fewer than 1% of intercontinental traffic
travels through satellites; the overwhelming majority moves through submarine
fibre cables, and there are around 600 of them.

Radio links — Wi-Fi, 4G, 5G — are real radio, with a shared medium and a
bandwidth budget that everyone on the link competes for. Your home Wi-Fi
connection is not "the internet" until something bridges it to a wired network.

This matters because the abstractions genuinely are layered. A TCP segment does
not travel; a modulated signal does, and the segment is recovered from it at
each hop. Understanding the boundary between "what the OS believes" and "what is
on the wire" is the point of this module.

## The packet-switching model

This is the single most important idea in the module.

### Store and forward

A **router** receives a packet, waits until it has the entire packet in its
buffer, looks up where it should go next, and sends it out that interface. It
holds the whole packet before forwarding any of it. This is **store and forward**,
and it is why routers have memory and why they can buffer.

The alternative, **cut-through** forwarding, starts transmitting as soon as the
destination address is known. It reduces latency by tens of microseconds and
increases complexity, because the router may have forwarded bytes to the wrong
link. Most core routers use store and forward.

```text
Store and forward:  [full packet received] -> [lookup] -> [transmit]
Cut through:                    [header read] -> [transmit as it arrives]
                                           ^
                                    lower latency, may pick wrong link
```

### Packets are independent

**This is the part that surprises people.** There is no connection. The internet
does not establish a session between two machines and then send data along it.
Every packet is routed **independently** from its source to its destination,
based only on its destination address.

The consequences are direct:

- Packets from one transfer may take different paths, and arrive out of order.
- Packets are lost routinely, and this is normal rather than exceptional.
- No router holds state about your traffic. Routing is per-packet, not
  per-conversation.
- A router can fail and its traffic re-routes, because there is no connection
  state to tear down.

TCP is what turns this unreliable, unordered mess into an ordered byte stream.
Until the TCP note, you should assume *no* guarantees at all.

### Why packet switching beat circuit switching

In a **circuit-switched** network, a dedicated path is reserved and stays
reserved for the duration of a call, whether or not it is carrying data. A
silent phone call holds a full-capacity path across the entire network.

Packet switching does not reserve anything. It interleaves packets from many
sources onto shared links and relies on statistical multiplexing — average
bandwidth demand is predictable even when individual demands spike, so the
aggregate fits in less capacity than the sum of the peaks.

Packet switching won for reasons that are mostly about cost and failure, not
efficiency:

| | Circuit switching | Packet switching |
|---|---|---|
| Capacity reserved | For the whole call | Only while data flows |
| Idle link | Wasted | Shared |
| Setup before data | Required | None |
| Failure impact | Path must be re-established | Packets re-route on their own |
| Cost of a new user | New circuit | Marginal |

That last row is what made the internet affordable at scale. Adding a user does
not require building capacity for them in advance.

The trade is that packet switching gives you **no guarantees**. The internet's
core makes a best-effort promise: it will try, it may drop your packet, it may
reorder it, and it will not tell you in advance what capacity you will get.
Predictable performance under load is achieved by overprovisioning and by
controlling congestion at the edges — not by the network core. This is why TCP's
congestion control exists and why it is the dominant force shaping traffic on the
internet.

### The datagram

The unit that routers forward is a **datagram**: an IP header plus payload, with
no connection state and no guarantee of anything. IPv4's header is 20 bytes
fixed, plus options. IPv6 simplified it to a fixed 40-byte header, dropping
options entirely and moving uncommon features into extension headers.

The header carries what a router needs to forward the packet and nothing more:

```text
IPv4 header, 20 bytes, min
  version(4) IHL(4)  DSCP(6) ECN(2)  total length(16)
  identification(16) flags(3) fragment offset(13)
  time to live(8)   protocol(8)   header checksum(16)
  source address(32)               destination address(32)
  [ options, if IHL > 5 ]

  payload: TCP segment, UDP datagram, ICMP message, ...
```

The `time to live` field is worth understanding because it explains `ping`
output. Every router decrements it by one before forwarding. At zero the packet
is discarded and an ICMP message is returned. Since the initial value bounds how
many hops a packet can survive, `ttl=57` in a `ping` reply means roughly 57 hops
of budget remained out of whatever the sender chose to start with — which is why
`ping` reveals your approximate distance in hops. The field is also a hard
loop-prevention mechanism: a misconfigured pair of routers bouncing a packet
between them forever is stopped by the TTL, not by good intentions.

## Layering, and what it costs

Protocols are arranged in layers, each using the services of the one below it
and providing services to the one above. This is **encapsulation**: as data
descends the stack at the sender, each layer prepends its own header; as it
ascends at the receiver, each layer strips its own header off.

```text
Application    HTTP GET /index.html
                 |
Transport      TCP  seq=1 ack=1 flags=PSH,ACK        <-- adds reliability
                 |
Network        IP   dst=203.0.113.10 ttl=63          <-- adds addressing
                 |
Link          Ethernet dst=aa:bb:cc:dd:ee:ff src=... <-- adds local delivery
                 |
Physical      0x41 0x45 ... voltage transitions on copper or fibre
```

Layering buys you **composability**: TCP does not need to know it is carrying
HTTP, and HTTP does not need to know whether it is carried over TCP or QUIC.
Each protocol can be replaced independently.

Layering costs you **headers**. Every layer adds overhead that the layer below
must transmit but not interpret. A `GET / HTTP/1.1` request is about 50 bytes of
payload, but it travels inside a 20-byte TCP header, a 20-byte IP header, and a
14-byte Ethernet header — and if the resulting frame is under 46 bytes of
payload it is padded out to the **64-byte Ethernet minimum frame size**. That
request costs well over 100 bytes of wire for 50 bytes of content.

This is a real cost you meet again in the TCP note with **Nagle's algorithm**,
which exists to avoid sending these small packets one at a time, and again
whenever you compute the bandwidth cost of many small requests.

The important boundary to keep in view: the **link layer is not part of the
internet**. An Ethernet frame is meaningful only between two machines on the same
link. It is stripped and replaced at every router. What survives the entire
journey is the IP datagram — and above it, the TCP segment, which is why TCP is
described as an end-to-end protocol even though the link layer is not.

## IP addressing is not identity

An **IP address identifies an interface on a link**, not a computer and not a
person. This distinction is the source of several things that surprise people.

An address is **topologically meaningful** — it encodes where in the network the
interface is, so routing can aggregate. `203.0.113.0/24` is a block of 256
addresses handed out to one network in one location. Addresses are allocated
hierarchically: IANA delegates to Regional Internet Registries, which delegate
to networks. See RFC 1918 for the private ranges, and note that
`192.0.2.0/24`, `198.51.100.0/24`, and `203.0.113.0/24` are reserved for
documentation and never routed.

Routing uses **longest-prefix match**: a packet is forwarded to the most
specific route that covers its destination. This is what makes aggregation work.
An ISP may announce `203.0.113.0/24`, and the rest of the internet may treat
that as a single route. If the ISP later announces `203.0.113.64/26` to a
neighbor, only routes for that quarter of the block change — everything else
still points at the aggregate. Without longest-prefix match, one network could
not be added to a region without updating every route in the world.

### NAT

Because an address identifies a location, addresses are consumable: a network
that grows needs more of them. **Network Address Translation (NAT)** breaks the
equivalence between address and location to stretch a scarce supply. A NAT
gateway rewrites the source address of outbound packets to its own public
address and keeps a table mapping internal connections to the port numbers it
chose, so replies can be routed back.

NAT solved address exhaustion and is now everywhere — in homes, in mobile
networks, in datacentre pods. Its costs are real:

- Inbound connections require explicit port forwarding or a hole-punching
  mechanism, so a device behind NAT is not reachable unless configured.
- The mapping is per-connection state, so NAT is stateful at scale, and stateful
  middleboxes are harder to run at line rate.
- End-to-end reachability is lost: two peers both behind NAT generally cannot
  connect directly without assistance.
- It violates the end-to-end principle from inside the network core.

**IPv6** removes the pressure: it allocates 128-bit addresses, enough that
giving every interface a globally routable address is practical. NAT is therefore
predicted to fade on the public internet while remaining useful inside
datacentres for policy and address management.

## The end-to-end principle

**The end-to-end principle**, articulated by Saltzer, Reed, and Clark in 1984,
states that a function can be implemented correctly only with the knowledge and
access to state that only the endpoints have, so placing it in the network core
is the wrong choice.

The argument is about correctness, not elegance. The endpoints can see the whole
message and know the application semantics. A router sees only a header. So a
transport mechanism that guarantees ordered, reliable delivery can be implemented
correctly only by the two processes that exchange the data — and, critically, a
correct implementation at the endpoints makes it correct regardless of what the
middle does.

This is why TCP lives at the endpoints and why the network core is deliberately
dumb. It is also why adding a new application protocol does not require
touching the network: any host can speak anything.

The principle is not absolute, and the interesting engineering comes from the
exceptions. Functionality placed in the middle for non-correctness reasons —
network address translation, firewalls, load balancers, deep packet inspection,
geo-DNS — is unavoidable rather than principled, and each one is a step away
from the ideal. This tension is not a historical curiosity; it is why encryption
in transit became a near-universal requirement after plaintext HTTP made passive
observation trivial.

## How to observe it yourself

Everything above is inspectable on your own machine with tools that are almost
certainly already installed. The commands below were run on a Linux host; output
is real, with addresses sanitized to RFC 5737 documentation ranges where the
original would leak real network details.

### 1. What interface am I on, and what address does it have?

```bash
ip -brief addr
```

```text
lo               UNKNOWN        127.0.0.1/8 ::1/128
wlp0s20f3        UP             192.168.1.71/24 2400:1a00:2b48:e7a9:bd69:f7b5:81ac:54ae/64
```

Two things to notice. The address is `192.168.1.71`, a **private** address from
the RFC 1918 range — not routable on the public internet, which is exactly why
your router needs NAT to talk to anyone. And the same interface holds both an
IPv4 and two IPv6 addresses, which is **dual-stack**: both protocols are live
simultaneously. Most connections today still use IPv4 even on dual-stack hosts.

```bash
ip route
```

```text
default via 192.168.1.254 dev wlp0s20f3 proto dhcp src 192.168.1.71 metric 600
192.168.1.0/24 dev wlp0s20f3 proto kernel scope link src 192.168.1.71 metric 600
```

Two routes, and they are the whole routing table. The `default` route — meaning
"anything else" — points at `192.168.1.254`, your gateway. This is the router
that performs NAT. `proto dhcp` means the route was installed by DHCP, the
protocol a host uses to request an address and gateway automatically at network
join time.

### 2. What is the maximum packet size?

```bash
ip link show | grep -o 'mtu [0-9]*'
```

```text
mtu 1500
mtu 65536
```

**MTU** — maximum transmission unit — is the largest packet that link will carry
without splitting it. `1500` is the Ethernet default and the value almost
every network in the path also uses, which is not a coincidence: it keeps
fragmentation from being needed. `65536` is the loopback interface's, and its
much larger value is why you can ping `127.0.0.1` with payloads that would be
rejected on a real link.

The consequence: a 1500-byte MTU with a 20-byte IP header and a 20-byte TCP
header leaves **1460 bytes** of payload. ICMP adds 8 bytes of its own header, so
a ping payload of 1472 bytes is the largest that fits in one 1500-byte packet —
which is exactly what `ping` reports when you ask it to send that much:

```text
PING 203.0.113.1 (203.0.113.1) 1472(1500) bytes of data.
```

The `1472(1500)` means 1472 bytes of payload totalling 1500 bytes on the wire.
One byte more and the total becomes 1501, exceeding your local MTU, and the
kernel refuses before anything leaves the machine:

```text
ping: sendmsg: Message too long
```

That error is the good case, and it is worth contrasting with what actually
happened on the real run behind this note. Sending 1472 bytes to a public host
produced **100% packet loss with no error at all**, while a 56-byte ping to the
same host succeeded:

```text
# 1472-byte payload
4 packets transmitted, 0 received, 100% packet loss, time 3065ms

# 56-byte payload, same host, immediately after
3 packets transmitted, 3 received, 0% packet loss, time 2002ms
```

Two entirely different failure signatures for the same operation, one byte-count
apart in what the kernel believed. The local kernel thought 1500 bytes was fine;
something further along the path disagreed, and the packet vanished without
complaint. This is the central MTU lesson, and it is genuinely counter-intuitive:
**a packet silently discarded is more dangerous than one actively rejected**,
because a rejected packet produces an error and a silent drop produces a
timeout.

This failure mode is not rare. Tunnels, PPPoE links, and some VPNs all reduce
the usable MTU. A connection that works for small requests and hangs forever for
large ones is almost always an MTU problem, and it is the reason **Path MTU
Discovery** exists. A router that cannot forward a packet should send an ICMP
"fragmentation needed" message back rather than dropping it silently, and the
sender then reduces its size. Note the security implication: an attacker who
blocks those ICMP messages can break connectivity in ways that are very hard to
diagnose from the application layer.

To reproduce the good case locally, compare against loopback, whose MTU of 65536
leaves room for payloads a real link would reject:

```bash
ping -c 1 -s 4000 127.0.0.1
```

### 3. Measure the round trip

```bash
ping -c 3 203.0.113.1
```

```text
PING 203.0.113.1 (203.0.113.1) 56(84) bytes of data.
64 bytes from 203.0.113.1: icmp_seq=1 ttl=57 time=9.20 ms
64 bytes from 203.0.113.1: icmp_seq=2 ttl=57 time=6.74 ms
64 bytes from 203.0.113.1: icmp_seq=3 ttl=57 time=9.48 ms

--- 203.0.113.1 ping statistics ---
3 packets transmitted, 3 received, 0% packet loss, time 2002ms
rtt min/avg/max/mdev = 6.739/7.808/9.203/1.031 ms
```

Read this carefully, because most of what people expect is not in it.

`56(84)` — 56 bytes of ICMP payload, 84 bytes total with the ICMP and IP
headers. The wire carries 84 bytes, then Ethernet framing pads the frame to 64
bytes minimum on the local link. Tiny packets are mostly overhead.

`ttl=57` — the returned TTL. The sender chose an initial value and each router
decremented it. The value that arrives back tells you roughly how many hops
remained, not how many there were, unless you know the initial value. This is
your approximate distance from the destination.

`time=9.20 ms` — **round-trip time**, the time for the ICMP echo request to
reach the host and the reply to come back. One-way latency is roughly half, but
only if the path is symmetric, which it frequently is not.

`0% packet loss` over 3 packets means almost nothing. **Three samples prove
nothing about loss rate.** Loss and latency vary by second in real networks; a
meaningful measurement needs hundreds of samples over time. Treat single-digit
ping output as a liveness check, not a measurement.

### 4. Watch the path, and watch it not be a path

```bash
tracepath -n -m 12 203.0.113.1
```

```text
 1?: [LOCALHOST]                      pmtu 1500
 1:  192.168.1.254                                         2.519ms
 2:  203.0.113.47                                        8.085ms
 3:  203.0.113.140                                       7.133ms
 4:  203.0.113.104                                      10.193ms
 5:  203.0.113.80                                        6.461ms
 6:  no reply
 7:  no reply
 8:  198.51.100.11                                       8.892ms asymm  7
 9:  no reply
10:  no reply
     Too many hops: pmtu 1500
```

This output is the best available argument for "packets are independent."

**The `no reply` entries are not missing hops.** A hop listed as `no reply`
still forwarded your packet. `tracepath` works by increasing a TTL and observing
who sends the ICMP time-exceeded message back. Many routers rate-limit or
deprioritize those messages, because generating them is work they would rather
not do on a full link. So the hop exists, handled your packet, and declined to
answer — which is entirely reasonable behaviour, not a failure.

This means **traceroute output is incomplete and the hop numbers are an upper
bound, not a count.** Some hops are always invisible. It also means the "asymm"
marker — here noting that hop 8 replied faster than hop 7 — is a real signal:
return paths are frequently not the reverse of forward paths. That asymmetry is
one reason round-trip time is not a clean measure of one-way latency.

`pmtu 1500` throughout means no router on this path needed to reduce the packet
size. If a smaller MTU existed ahead, `tracepath` would have found it and
reported it — this tool discovers MTU along the path as a side effect of how it
works.

### 5. See the network make a decision

```bash
ip route get 203.0.113.1
```

```text
203.0.113.1 via 192.168.1.254 dev wlp0s20f3 src 192.168.1.71 uid 1000
    cache
```

Without sending anything, this asks the kernel's routing decision for that
destination: which interface, which next hop, which source address it would
pick. When a connection mysteriously cannot be established, this is often the
fastest way to find out why — a wrong route, a wrong source address, or a policy
rule shows up here rather than as a timeout 30 seconds later.

### 6. Confirm nothing is a connection

```bash
dig +noall +answer example.com
```

```text
example.com.		175	IN	A	172.66.147.243
example.com.		175	IN	A	104.20.23.154
```

DNS is an application-layer protocol, not part of the internet's core — it runs
over UDP like any other program. What makes it foundational is what it returns:
an IP address. Everything downstream needs an address before it can send
anything, which is why DNS resolution is on the critical path of every connection
you have ever made. It gets its own note.

### 7. See connections in the kernel's view

```bash
ss -tnp
```

```text
State  Recv-Q Send-Q   Local Address:Port      Peer Address:Port   Process
ESTAB  0      0         192.168.1.71:36868      198.51.100.32:443   users:(("brave",pid=2385,fd=38))
ESTAB  0      0         127.0.0.1:49374        127.0.0.1:44768     users:(("opencode",pid=5627,fd=29))
ESTAB  0      0         192.168.1.71:50142     198.51.100.26:443   users:(("brave",pid=2385,fd=27))
```

Real output, with the public peer addresses replaced by the RFC 5737
documentation ranges. Local addresses, ports, and PIDs are unmodified.
Note the first column: `ESTAB` is **established**, and it is one of several
states a TCP socket moves through. A connection being created sits in `SYN-SENT`
while the handshake is in progress. Watching those states change is the clearest
evidence that connection setup is a real exchange of messages rather than an
instantaneous act. It is the same three-way handshake the TCP note covers, and it
is why a new TCP connection costs a full round trip before any application data
can move — one of the first things HTTP/2 multiplexing and HTTP/3's connection
coalescing exist to avoid.

Also note `Recv-Q` and `Send-Q` at `0`: those are bytes queued for the local
process to read and for the kernel to send. They are the socket buffer queues,
and when they stop being zero you have found your bottleneck or a stalled peer.
The `Process` column maps each socket to the program holding it, which is the
quickest way to answer "what in this machine is talking to that address".

### What to try next

- Run `ping -c 100` to a host in your city and in another country, and compare
  the distributions. Notice that latency is a floor set by physics, while
  bandwidth is a capacity you pay for.
- Compare `ping` to your default gateway against `ping` to a host across the
  ocean, and account for the difference.
- On a dual-stack host, run `curl -4 -v` and `curl -6 -v` against the same host
  and compare which protocols are actually used and how the connections differ.
- Repeat the MTU experiment yourself: `ping -c 2 -s 1472` and `ping -c 2 -s
  1473` against a public host, then the same against `127.0.0.1`. Try to find a
  payload size where the loopback works and the remote host does not, and reason
  about which link in the path disagreed.
- Watch a socket transition: run `ss -tn state syn-sent` in a loop while you
  open a connection to a host that will not respond, and watch `SYN-SENT`
  persist. That state persisting *is* the timeout.

## Common misconceptions

**"The internet is a cloud."**
No. "The cloud" is a marketing term for someone else's datacentre. The internet
is specific, physical infrastructure — submarine cables, fibre, routers in
buildings. Latency to a given city is dominated by geography and by how many
network hops sit between you, and neither is hidden by the metaphor.

**"A packet travels from A to B."**
It arrives at B. It does not travel. Each router handles it independently and
forwards it based on a destination address. Packets in the same transfer may take
different routes and arrive out of order.

**"An IP address identifies a computer."**
It identifies an interface on a link, and it encodes location. One host has many
interfaces and many addresses. NAT adds a further layer of indirection where one
public address can represent many internal hosts.

**"Pings measure the speed of the internet."**
Ping measures round-trip time to one host over one path at one moment. It says
nothing about bandwidth, nothing about throughput, and little about loss over
time. `ping` output with a handful of samples is a liveness check.

**"Bandwidth is how fast the internet is."**
Bandwidth is capacity — bytes per second. Latency is delay — seconds. Throughput
is what you actually achieve. You can have enormous bandwidth with terrible
latency, or modest bandwidth with very low latency. A 10 Mbps link with 5 ms
latency feels far more responsive than a 1 Gbps link with 200 ms latency,
especially for interactive work.

**"TCP is part of the internet."**
TCP is one application of the internet's transport layer, running at the
endpoints. The internet's core is IP. HTTP/3 replaces TCP with QUIC over UDP,
and the internet is largely unaffected — a direct demonstration of the
end-to-end principle in action.

**"More hops always means more latency."**
Hop count and latency are related but not the same. Each hop adds a minimum of
processing and propagation delay, but a path with more hops through well-placed
peering can beat a shorter path through congested transit. This is exactly why
BGP supports policy-based rather than purely shortest-path selection.

## Trade-offs

The internet's design is a set of compromises, and each one costs something
specific.

**No quality of service in the core.** Packets are treated alike, so there is no
way to reserve capacity for a high-priority flow. Latency-sensitive traffic
competes on equal terms with bulk transfers. This is why real-time media and
gaming rely on application-level adaptation — FEC, jitter buffers, adaptive
bitrate — instead of network-level priority.

**State lives at the edges.** Because routers hold no per-connection state, the
core scales to enormous throughput with modest hardware. The cost is that
reliability, ordering, and congestion control must be implemented by every
endpoint, duplicated across every protocol. A new transport means reimplementing
them.

**Best-effort means you cannot buy a guarantee.** Capacity planning is
overprovisioning plus measurement. Two networks on the same path have no
guaranteed relationship to each other's performance.

**NAT, firewalls, and middleboxes erode the end-to-end model.** Each is placed
in the core for non-correctness reasons and each costs reachability, state, or
observability. The long-term direction is to move function back to the endpoints
where it can be implemented correctly — TLS in the client, encryption at rest in
the process — rather than relying on perimeter filtering.

**Decentralization slows change.** With no central authority, coordination happens
through rough consensus and running code. This produces a stable, boring core
that has barely changed in decades, and a slow, contentious edge where new
protocols and policies get negotiated. That is the correct trade for the core,
and a real cost at the edge.

## Related concepts

- [How data transfers](./02-how-data-transfers.md) — encapsulation, ARP, and why
  the IP address stays constant while the MAC address changes at every hop.
- [TCP](./04-tcp.md) — where reliability, ordering, flow control, and congestion
  control actually get implemented.
- [DNS](./03-dns.md) — how a name becomes the address every other protocol
  depends on.
- [UDP](./05-udp.md) — the transport layer without any reliability machinery.
- NAT, and what breaks when a middlebox sits in the path — covered above, and
  expanded in the TCP note.
- The request lifecycle — module 02, not yet written.

## References

- RFC 791 — Internet Protocol (STD 36). The IPv4 specification.
  https://www.rfc-editor.org/rfc/rfc791
- RFC 8200 — Internet Protocol, Version 6 (IPv6) Specification, STD 86.
  https://www.rfc-editor.org/rfc/rfc8200
- RFC 1122 — Requirements for Internet Hosts (STD 3). What every host must do.
  https://www.rfc-editor.org/rfc/rfc1122
- RFC 1918 — Address Allocation for Private Internets.
  https://www.rfc-editor.org/rfc/rfc1918
- RFC 5737 — IPv4 Address Blocks Reserved for Documentation. The sanitized
  ranges used in the commands above. https://www.rfc-editor.org/rfc/rfc5737
- RFC 2328 — OSPF Version 2. An interior gateway protocol, for contrast with
  BGP. https://www.rfc-editor.org/rfc/rfc2328
- RFC 4271 — BGP-4. The protocol that connects autonomous systems.
  https://www.rfc-editor.org/rfc/rfc4271
- RFC 7606 — Recommendations on Transport of Path MTU Discovery.
  https://www.rfc-editor.org/rfc/rfc7606
- RFC 9293 — Transmission Control Protocol (STD 7). The TCP specification.
  https://www.rfc-editor.org/rfc/rfc9293
- D. Saltzer, D. Reed, D. Clark, "End-to-End Arguments in the Design of Computer
  Networks", ACM SIGCOMM Computer Communication Review, 1984.
  https://doi.org/10.1145/3149.1661
- V. Jacobson, "Congestion Avoidance and Control", ACM SIGCOMM, 1988.
- B. Carpenter, L. Peterson, "Building a Network: The Road to the Internet",
  ACM Queue, 2012. https://queue.acm.org/detail.cfm?id=2142069
- J. Kurose, J. Ross, *Computer Networking: A Top-Down Approach*, 8th ed.,
  Pearson, 2021, ch. 1. Useful as a contrast: it starts from the application
  layer, this module starts from the substrate.
- Linux man-pages: `ip(8)`, `ping(8)`, `tracepath(8)`, `ss(8)`.
  https://man7.org/linux/man-pages/
