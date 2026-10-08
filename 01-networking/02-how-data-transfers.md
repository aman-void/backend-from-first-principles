---
title: How Data Transfers
module: 01-networking
status: draft
prerequisites:
  - ./01-all-about-the-internet.md
---

# How Data Transfers

The previous note described the internet's structure and the packet-switching
model. This note follows a single request from the moment an application calls
`write()` to the moment the bytes come back, and shows every layer being added
and removed along the way.

The question this note answers is one that is easy to skip and expensive to get
wrong: **when you send data to a machine you are not directly connected to, what
address do you actually put on the packet, and who does the addressing?**

The short answer, which most people never hear stated plainly:

> The **IP address stays the same for the entire journey**. The **MAC address
> changes at every single hop.**

Almost every networking mystery traces back to not knowing this.

## Why does this exist?

The previous note's mental model — routers forwarding packets independently —
leaves a real gap. IP addresses identify an interface somewhere in the world, and
a router cannot know which of possibly millions of machines on a link you meant.
It also cannot send a frame to an IP address, because frames are addressed by MAC
address and routers do not look inside frames.

So there are two addressing systems, at two different layers, doing different
jobs:

- **IP addresses are end-to-end and hierarchical.** They encode location, so
  routing can aggregate and forward toward a region. They survive every hop.
- **MAC addresses are flat and local.** They identify a specific interface on one
  link. They mean nothing beyond that link and are discarded at every router.

This note covers how a local frame gets built around an IP packet, how the
destination MAC is discovered, and how a router tears the frame off and builds a
new one for the next hop.

## The two address families

### MAC addresses

A **MAC address** is 48 bits, conventionally written as six hex bytes
(`b4:1d:62:bc:e4:00`). The first 24 bits identify the manufacturer, the rest
identify the device. It is a **flat** identifier — no structure, nothing to route
on.

This is the key property: a MAC address has no location encoded in it. Your
laptop's MAC and your neighbour's MAC differ arbitrarily, with no relationship
between the two. You cannot build a routing table from MAC addresses, because
there is nothing to aggregate.

Confirm yours:

```bash
ip -brief link show
```

```text
lo               UNKNOWN        00:00:00:00:00:00 <LOOPBACK,UP,LOWER_UP>
enp1s0           DOWN           14:16:9e:82:c7:21 <NO-CARRIER,BROADCAST,MULTICAST,UP>
wlp0s20f3        UP             8c:1d:96:99:4a:62 <BROADCAST,MULTICAST,UP,LOWER_UP>
```

`14:16:9e` is OUI. Yours will differ; that is the point — the prefix identifies
the vendor, and there is no locality encoded anywhere.

### IP addresses

An IP address is hierarchical and topologically meaningful. `192.168.1.71/24`
tells you three things without any lookup: this is a private address, it is in a
`/24` block, and the first 24 bits (`192.168.1`) identify the network while the
last 8 identify a host within it. That structure is exactly what lets a router
forward your packet toward the right part of the world.

See [All about the internet](./01-all-about-the-internet.md) for allocation and
routing. The relevant point here is the contrast: **one address family is
designed for aggregation, the other is not.**

## Encapsulation: building the frame

When your kernel sends data, it does not just write the payload. It wraps it
repeatedly, once per layer, and each wrapper is only meaningful to the layer that
added it.

Starting from an HTTP request, here is what the sender builds:

```text
  Application   GET / HTTP/1.1\r\nHost: example.com\r\n\r\n
                    |
                    |  layer 4 adds ports
                    v
  Transport     | src port 34567 | dst port 443 | TCP header |
                    |
                    |  layer 3 adds IP addresses
                    v
  Network       | src 192.168.0.2 | dst 203.0.113.10 | IP header |
                    |
                    |  layer 2 adds MAC addresses for THIS link
                    v
  Link          | src b4:1d:62:bc:e4:00 | dst aa:bb:cc:dd:ee:ff | Ethernet |
                    |
                    |  layer 1 puts it on the medium
                    v
  Physical      voltage transitions on copper, light in fibre, RF in air
```

Each header serves one purpose and is discarded once it has done that job. The
Ethernet header exists so that **this one link** can deliver the frame; the router
at the end of the link does not need it and does not read it.

### What each address is for

```text
  field        scope            changes?            used by
  -----------  ---------------  ------------------  ----------------
  MAC src/dst  one link         at EVERY hop        senders + switch
  IP src/dst   end to end       NEVER               every router
  port src/dst end to end       NEVER               TCP/UDP at endpoints
```

Read that table again. The IP address and the ports are end-to-end and survive the
whole journey. The MAC addresses are rewritten at every hop. This is the single
most useful thing to internalize from this note.

## ARP: finding the MAC address you need

You now have the situation: you want to send a packet to `203.0.113.10`, but you
can only send frames to a MAC address, and you do not know it.

**ARP — the Address Resolution Protocol — is how you find out.** It asks, on the
local link, "who has this IP address? Tell me your MAC address."

The request is a **broadcast**, because you do not know which machine to ask:

```text
  request:  "Who has 192.168.0.7? Tell 192.168.0.2."
  sent to:  ff:ff:ff:ff:ff:ff   (broadcast)
```

Every machine on the link receives it, and only the one holding that address
replies — as a **unicast** to the asker:

```text
  response: "192.168.0.7 is at 4d:d4:fd"
  sent to:  the MAC of the machine that asked
```

That two-message exchange is the whole protocol. Here is the full sequence from
a local network, with the encapsulation stack on the left showing what is being
built at each step:

```text
  Building the frame          Local network 192.168.0.0/24

  layer 4  | src 34567 dst 443  |      .2 ──────┐      ┌──── .1
          | TCP header          |              │      │    (gateway)
  layer 3  | src .2  dst .7     |              │      │
          | IP header           |      .5 ─────┼──────┤
          |                     |              │      │
  layer 2  | src ?   dst ?      |      .7 ─────┘      └────
          | Ethernet            |      (server, port 443)
```

```text
  Step 1  .2 has no MAC for .7. It broadcasts:

          "Who is 192.168.0.7? Send me your MAC address."
               │
               ├──────────────► .1  (ignores, not .7)
               ├──────────────► .5  (ignores, not .7)
               └──────────────► .7  (matches!)

  Step 2  .7 replies, unicast to .2:

          "This is my MAC address: 4d:d4:fd"
               ◄─────────────── .7

  Step 3  .2 caches it and now builds the real frame:

          | src .2 MAC    dst 4d:d4:fd |  | src .2 dst .7 |  | ports |
```

The broadcast goes to every machine because the link layer has no addressing
scheme for "the one with IP 192.168.0.7" — MAC addresses are flat, and the
mapping between them and IPs is exactly what ARP is being asked to discover.

### The ARP cache

Answering that broadcast for every single packet would be absurdly wasteful, so
the mapping is cached:

```bash
ip neigh show
```

```text
192.168.1.254 lladdr b4:1d:62:bc:e4:00 REACHABLE
fe80::b61d:62ff:febc:e400 lladdr b4:1d:62:bc:e4:00 router REACHABLE
2400:1a00:2b48:e7a9:b61d:62ff:febc:e400 lladdr b4:1d:62:bc:e4:00 router REACHABLE
```

Real output from a Linux host. Note:

- The first entry is the **gateway's** MAC, which is why essentially every outbound
  packet populates this table with the same entry.
- `REACHABLE` is a state, not just a boolean. ARP entries expire, because a
  machine's MAC can change (new NIC, virtual machine moved, DHCP reassignment) and
  a stale entry sends your frames into the void.
- The IPv6 entries show that **ARP is not the whole story**. IPv6 replaces ARP with
  **NDP** (Neighbor Discovery Protocol, RFC 4861), which uses ICMPv6 and supports
  router discovery. Same purpose, different protocol.

Flush the cache and watch it repopulate:

```bash
sudo ip neigh flush dev wlp0s20f3
ip neigh show
```

Empty table. Ping the gateway, and it returns:

```bash
ping -c 1 192.168.1.254
ip neigh show dev wlp0s20f3
```

This is a satisfying experiment because you can see the cache go from empty to
populated within milliseconds of the ping, and it makes the mechanism concrete
rather than abstract.

## The part that surprises everyone: when the destination is not local

Now the crucial case. You want to reach `203.0.113.10`, which is **not** in your
`192.168.0.0/24` subnet. You do not know its MAC address — it is not on your
link, it is on a different network, and its MAC is a private matter between it
and whatever router is next to it.

You cannot ARP for it. But **the IP address on the packet is still the real
destination.** Your host builds the packet with `dst 203.0.113.10` and addresses
the frame to the **gateway's MAC**:

```text
  Packet (layer 3)              Frame (layer 2)
  ┌──────────────────────┐      ┌──────────────────────────────┐
  │ src 192.168.0.2      │      │ src b4:1d:62:bc:e4:00        │
  │ dst 203.0.113.10  ◄──┼─ IP ─┼─► dst aa:bb:cc:dd:ee:ff      │
  │ TCP: 34567 -> 443    │      │      (the gateway's MAC)     │
  └──────────────────────┘      └──────────────────────────────┘
                                   ^
                          destination on the WIRE is the gateway
                          destination in the PACKET is the server
```

This is the whole trick. **The MAC address says "who on this link should take
this frame"; the IP address says "who this packet is ultimately for."** They
answer different questions, and for an off-subnet destination they point at
different machines.

Your host decides local versus remote by subnet mask. `/24` means "the first 24
bits are network, so anything sharing `192.168.0` is local." Everything else goes
to the default route.

Ask the kernel its decision without sending anything:

```bash
ip route get 8.8.8.8
```

```text
8.8.8.8 via 192.168.1.254 dev wlp0s20f3 src 192.168.1.71 uid 1000
    cache
```

`via 192.168.1.254` is the gateway, `dev wlp0s20f3` the interface, `src` the
source address the kernel will use. This single command tells you exactly what
the host is about to do, and it is the fastest way to diagnose "why can't I reach
this host."

## What a router does to your packet

The router at the end of the link receives the frame, and performs the core act of
routing. Here is the sequence, and it is where the whole note comes together:

```text
  Host .2                Router 1              Router 2            Server .7
    |                       |                      |                    |
    |  frame dst = R1 MAC   |                      |                    |
    |---------------------->|                      |                    |
    |                       |                      |                    |
    |                       |  1. strip L2 header  |                    |
    |                       |     IP dst = .7      |                    |
    |                       |  2. lookup route     |                    |
    |                       |     .7 is via R2     |                    |
    |                       |  3. decrement TTL    |                    |
    |                       |  4. ARP for R2  (if  |                    |
    |                       |     not cached)      |                    |
    |                       |  5. build NEW frame: |                    |
    |                       |     IP dst still .7  |                    |
    |                       |     MAC dst = R2     |                    |
    |                       |--------------------->|                    |
    |                       |                      |                    |
    |                       |         ... same three steps ...          |
    |                       |--------------------->|                    |
    |                       |                      |  frame dst = .7 MAC |
    |                       |                      |------------------->|
    |                       |                      |                    |
    |                       |     IP dst = .7 the whole way            |
```

Watch what is constant and what changes:

**Constant across every hop:** source IP, destination IP, source port,
destination port, TCP sequence numbers, and all the payload. The router does not
touch any of it. It has no idea what the payload means.

**Changed at every hop:** source and destination MAC addresses, the Ethernet type
field, the frame check sequence, and **the TTL**.

The IP header is opened, read, and rewritten — but only two fields change: the
TTL and the header checksum that covers it. The source and destination addresses
are **never modified by a normal router.**

### Why the TTL matters

Each router decrements the TTL by one before forwarding. At zero, the packet is
discarded and an ICMP time-exceeded message is sent back.

This is not bookkeeping for its own sake. It exists because misconfigured routers
would otherwise bounce a packet between them forever, and one loop would saturate
the links involved. The TTL is the loop-breaker.

It also gives you your distance in hops, which is why `ping` and `traceroute`
reveal topology at all. See
[All about the internet](./01-all-about-the-internet.md) for what a returned TTL
value does and does not tell you.

### What the router deliberately ignores

A router reads the IP header. It does not read the TCP header, does not read the
HTTP request, and has no idea the payload is a GET for `/`.

This is the **end-to-end principle** in physical form. Because routers do not
interpret the payload, you can run any protocol over the internet without asking
anyone's permission — that is exactly why adding HTTP/3 or QUIC required no
changes to the network. The cost is equally real: the network cannot prioritise,
authenticate, or rate-limit on the basis of what the data *is*.

## Store and forward, revisited

Each router waits until it has the **entire** packet before forwarding any of it,
then forwards it on a single outgoing link.

This is why a router's memory matters: it must buffer packets arriving faster than
it can send them. When a link's output is congested, queues fill, and the excess
is **dropped** — silently, with no notification to the sender.

That silence is the source of TCP retransmissions. TCP cannot distinguish "lost
because the router was busy" from "lost because it crossed an ocean," so its only
remedy is to send it again, and that is what makes a congested network
congested. This is why congestion control exists at the endpoints at all: the
network core has no way to manage congestion, so it delegates it to whoever is
generating the traffic.

## The full journey, end to end

```text
  SENDER HOST              ROUTER 1        ROUTER 2         DESTINATION
  ───────────              ────────        ────────         ───────────

  app: GET /
    │
    ├─ layer 4:  add src port 34567, dst port 443
    │
    ├─ layer 3:  add src 192.168.0.2, dst 203.0.113.10, ttl=64
    │
    ├─ ARP:      who has MAC of gateway?  ──► cache
    │
    ├─ layer 2:  add src <my MAC>, dst <gateway MAC>
    │
    └─ send ═══════════════► strip L2, read dst IP
                             ttl 63, route lookup
                             ARP for R2 MAC
                             build frame: dst <R2 MAC>
                             IP hdr UNCHANGED ═══════► strip L2, read dst IP
                                                          ttl 62, route lookup
                                                          ARP for host MAC
                                                          build: dst <host MAC>
                                                          IP hdr UNCHANGED
                                                                             │
                                                          strip L2, deliver  │
                                                          ┌──────────────────┘
                                                          ▼
                                                    deliver to port 443
                                                    strip L3, hand to TCP
                                                          │
                                                          ▼
                                                    TCP reassembles,
                                                    app gets bytes
```

The IP header is copied forward eleven times and modified twice per pass. The
MAC addresses are created, used, and thrown away at every single link. That
asymmetry is the entire design.

## Practical experiment: watch it happen

Capture what actually crosses the wire. `tcpdump` needs elevated privileges for a
raw socket:

```bash
# watch ARP as the cache repopulates
sudo tcpdump -i wlp0s20f3 -n -e arp

# in another terminal
sudo ip neigh flush dev wlp0s20f3
ping -c 1 192.168.1.254
```

You will see the broadcast request arrive, the unicast reply, and the cache
repopulate. If `tcpdump` reports `You don't have permission to perform this
capture`, that is missing `CAP_NET_RAW` — the reason it needs root.

Watch the headers being added and stripped:

```bash
# filter for one host, show link and network layers
sudo tcpdump -i wlp0s20f3 -n -e -vv host 8.8.8.8 &
ping -c 2 8.8.8.8
```

`-e` prints the link-layer header. Compare what you see for the outgoing request
against the incoming reply: **the IP addresses in both directions are yours and
`8.8.8.8`, but the MAC addresses are the gateway's in both directions** — because
every frame between you and that host is addressed to your gateway.

Trace the path and see the hop count:

```bash
tracepath -n 8.8.8.8
```

Each line is one router decrementing your TTL and reporting it. Gaps — `no reply`
— are routers that forwarded your packet but chose not to answer, which is normal
and not a failure.

## Common misconceptions

**"The MAC address identifies my computer."**
It identifies your network interface. A laptop has one MAC per interface, a server
with four NICs has four, and virtual machines each have their own. The IP address
has the same problem, for the same reason: addresses belong to interfaces, not to
machines.

**"The destination MAC is the destination machine's MAC."**
Only when it is on your link. For anything off-subnet, the destination MAC is your
**gateway's**, while the destination IP stays the real target. Conflating these is
the root of most "why does my traffic show the router's MAC address" confusion.

**"Routers rewrite the source IP address."**
They do not. Normal forwarding modifies TTL and the header checksum, and nothing
else. Source rewriting happens in NAT — a specific, deliberately stateful function,
not routing.

**"ARP works across the internet."**
It does not, and cannot. ARP broadcasts are confined to a single link; routers do
not forward them. Every IP-to-MAC lookup you ever do is for a machine on your local
network or your next hop.

**"More hops means more latency."**
Each hop adds some processing delay, so it correlates. But a path with more hops
through good peering can beat a shorter path through congested transit, because
hop count measures distance and not load.

**"Switches are routers."**
A switch forwards using **MAC addresses** and stays in layer 2. A router forwards
using **IP addresses** and moves between layer 3 networks. Your home router is
usually both devices in one box.

## Trade-offs

**Two address families instead of one.** A single universal address would be
simpler, but IP addresses are too scarce and too structured to identify a specific
interface on a specific link, while MAC addresses are too flat to route on. The
duplication is not redundancy — each does a job the other cannot.

**ARP broadcasts scale only within a link.** The protocol is trivially simple and
perfectly adequate for a subnet, but it is a broadcast, so it cannot work across a
routed boundary. Larger layer-2 domains therefore need IPv6's NDP or other
workarounds.

**Per-hop MAC rewriting costs nothing and buys isolation.** A router that knows
nothing about TCP can forward TCP, UDP, QUIC, or anything else, and can be
replaced without renegotiating with any endpoint. This is why the core network
stayed stable while everything above it was replaced.

**The Ethernet minimum frame size wastes bandwidth.** The minimum frame is 64
bytes measured **from the start of the destination MAC to the end of the frame
check sequence**, which includes the 14-byte Ethernet header and the 4-byte FCS.
So the smallest payload that avoids padding is `64 − 14 − 4 = 46` bytes:

```text
  minimum frame                     64 bytes
   - Ethernet header (dst+src+type) -14
   - frame check sequence           - 4
   ----
  minimum payload                    46 bytes   anything less gets padded
```

A 20-byte payload therefore occupies a full 64-byte frame — **more than half the
frame is padding.** This is a legacy concession to collision detection, which no
longer exists on switched networks, and it is a real cost in request-heavy
workloads. Note the compounding: small TCP segments pay this padding *and* a
20-byte TCP header on every link.

**IPv6 removes ARP but adds its own complexity.** NDP handles neighbour discovery
and router discovery, but introduces extra protocol messages, and its use of
ICMPv6 has been a recurring source of broken middleboxes and firewalls.

**The end-to-end principle cuts both ways.** Routers ignoring payload enables
innovation and cheap core hardware, but it also means the network cannot protect
you. Filtering, rate limiting, and traffic shaping all require inspecting what you
are carrying — which pushes functionality to the edges, where it can be bypassed.

## Related concepts

- [All about the internet](./01-all-about-the-internet.md) — IP addressing,
  routing, longest-prefix match, and NAT. Read this first.
- [TCP](./04-tcp.md) — what happens to the port numbers as the packet travels.
- [UDP](./05-udp.md) — the same journey with a connectionless transport.
- [DNS](./03-dns.md) — how you find the destination IP address before any of this
  begins.
- Ethernet and switching — the layer 2 machinery that delivers frames on a link.
- IPv6 and NDP — how neighbour discovery works without ARP.
- NAT — the one common case where a router does rewrite addresses, and why.

## References

- RFC 826 — An Ethernet Address Resolution Protocol: A Protocol for Address
  Resolution. The entire specification of ARP, in one page.
  https://www.rfc-editor.org/rfc/rfc826
- RFC 4291 — IP Version 6 Addressing Architecture, §2.1. Why addresses belong to
  interfaces. https://www.rfc-editor.org/rfc/rfc4291
- RFC 4861 — Neighbor Discovery for IPv6. IPv6's replacement for ARP.
  https://www.rfc-editor.org/rfc/rfc4861
- RFC 791 — Internet Protocol, §3.1. IPv4 header format and TTL processing.
  https://www.rfc-editor.org/rfc/rfc791
- RFC 1122 — Requirements for Internet Hosts, §3.3.1. Host forwarding rules.
  https://www.rfc-editor.org/rfc/rfc1122
- IEEE 802.3 — Ethernet. The link-layer specification, including the 64-byte
  minimum frame size. Not freely available; the summary in
  A. Tanenbaum, *Computer Networks*, 6th ed., ch. 5, is an accessible substitute.
- A. Tanenbaum, D. Wetherall, *Computer Networks*, 6th ed., Pearson, 2022, ch. 5.
- Linux man-pages: `ip(8)`, `arp(8)`, `tcpdump(8)`, `net.ipv4.ip(7)`,
  `netdevice(7)`. https://man7.org/linux/man-pages/
