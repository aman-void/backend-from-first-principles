---
title: DNS
module: 01-networking
status: draft
prerequisites:
  - ./01-all-about-the-internet.md
  - ./02-how-data-transfers.md
---

# DNS

The Domain Name System is a distributed, hierarchical, cached database that maps
human-readable names like `example.com` to IP addresses like `203.0.113.10`. It
is an application-layer protocol, not part of the internet's core — it runs over
UDP like any other program — and it sits on the critical path of essentially every
connection you have ever opened.

That last point is what makes DNS worth understanding properly. Before you can
open a TCP connection, you need an address. Before you can send a UDP packet to
a remote host, you need an address. DNS is the first thing that happens, it is
invisible when it works, and when it is slow or wrong it is often misdiagnosed
as something else entirely.

## Why does this exist?

Because IP addresses are unusable by humans and names are unusable by routers.

`203.0.113.10` encodes location and nothing else. You cannot infer who operates
it, whether it is a web server or a printer, or whether a similar address
belongs to the same organization. Meanwhile `example.com` is meaningful to people
but means nothing to a router — the packet forwarding algorithm needs a number.

So we need both, plus a translation between them. The obvious design is a
central directory. That was actually built, and it did not scale:

- A single directory must contain every name, so its size and availability become
  everyone's problem. One outage halts name resolution globally.
- Loading a complete directory on every host is impractical.
- Organizations need to change their own records without coordinating with a
  central operator.

DNS solves this with **delegation**. The database is split into a tree, and each
part is managed by whoever owns that part of the namespace. No one holds the whole
thing, and no one has to be trusted to hold it.

## The hierarchy

The namespace is a tree, read right to left from the root:

```text
  .                     root zone          (IANA coordinates)
  |
  +-- com                TLD                (registry operator)
  |     |
  |     +-- example.com                  (registrant / authoritative)
  |           |
  |           +-- www.example.com          (DNS records, possibly same server)
```

Four roles, and confusing them is the most common DNS misunderstanding:

| Role | What it does | Who operates it |
|---|---|---|
| **Root server** | Knows which TLD servers exist for every TLD | 12 organizations, 13 named identities |
| **TLD server** | Knows which servers are authoritative for each domain in one TLD | Registry operator (Verisign for `.com`, RIPE NCC for `.ripe`, etc.) |
| **Authoritative server** | Holds the actual records and answers definitively | The domain owner |
| **Resolver** | Queries the hierarchy on a client's behalf and caches the answer | Your ISP, or a public resolver |

**The root server does not know any website's IP address.** It knows only which
servers handle `.com`. The common mental image of "ask the root server, it gives
you the address" is wrong, and understanding why is most of what there is to
understand about DNS.

### The thirteen root servers

There are **13 named root server identities** — `a.root-servers.net` through
`m.root-servers.net` — but not 13 machines. Because anycast routing allows one
address to be served from many locations, these 13 identities are served by
hundreds of physical instances in data centres worldwide, anycast-routed so that
queries are answered by the nearest one. Twelve organizations operate them
(Verisign operates two, `a` and `j`).

The count of 13 is historical accident: it was chosen in 1987 to fit in a UDP
packet alongside the original DNS query. See RFC 1035.

```bash
dig +noall +answer . NS
```

```text
.			518399	IN	NS	a.root-servers.net.
.			518399	IN	NS	b.root-servers.net.
.			518399	IN	NS	c.root-servers.net.
.			518399	IN	NS	d.root-servers.net.
.			518399	IN	NS	e.root-servers.net.
.			518399	IN	NS	f.root-servers.net.
.			518399	IN	NS	g.root-servers.net.
.			518399	IN	NS	h.root-servers.net.
.			518399	IN	NS	i.root-servers.net.
.			518399	IN	NS	j.root-servers.net.
.			518399	IN	NS	k.root-servers.net.
.			518399	IN	NS	l.root-servers.net.
.			518399	IN	NS	m.root-servers.net.
```

Count them: 13 named authorities. Note that the TTL here is 518399 — seven days —
which tells you the root zone changes rarely and heavily cached data should be
trusted.

## How a resolution actually happens

Your machine does not walk the hierarchy itself. It asks a **resolver**, which
does the walking on its behalf and caches everything it learns.

```text
  [your host]  -->  [resolver]  -->  [root]  -->  [TLD]  -->  [authoritative]
                       ^
                  your stub resolver asks this one
```

This is **recursion** for the query, **referral** for the answers. The distinction
matters when you are debugging:

- Your stub resolver sends a **recursive** query to the resolver: "give me the
  answer, do whatever it takes."
- The resolver sends **non-recursive** queries down the hierarchy. Each server
  answers from what it knows and, critically, **does not query on the resolver's
  behalf**. Instead it returns a *referral*: "I don't know the address, but ask
  these servers."

Each referral is one step closer to the answer, and the resolver walks down until
an authoritative server answers.

```text
   host             resolver           root             TLD            auth
    |                  |                 |                |             |
    |-- "example.com?"->|                 |                |             |
    |                  |-- "example.com?"->|                |             |
    |                  |<--- referral: .com servers ------|             |
    |                  |                 |                |             |
    |                  |                 |-- "example.com?"->|             |
    |                  |<-------- referral: example.com nameservers --|
    |                  |                 |                |             |
    |                  |                 |                |-- query --> |
    |                  |<---------------------- answer: 203.0.113.10 --|
    |<--- 203.0.113.10-|                 |                |             |
```

**Nothing returns all the way back to your host except the resolver's final
answer.** The root and TLD servers talk only to the resolver. This matters
operationally: it means the authoritative server for your domain sees queries
from resolvers, not from end users, so your access logs are full of resolver
addresses and cannot be used to count visitors.

### Following the real thing

`dig +trace` starts from the root and shows each step, which is the fastest way to
see this hierarchy rather than infer it:

```bash
dig +trace example.com
```

```text
; <<>> DiG 9.20.27 <<>> +trace example.com
;; global options: +cmd
.			518399	IN	NS	a.root-servers.net.
.			518399	IN	NS	b.root-servers.net.
...   (13 NS records)
;; Received 936 bytes from 127.0.0.53#53(127.0.0.53) in 409 ms

com.			172800	IN	NS	a.gtld-servers.net.
com.			172800	IN	NS	b.gtld-servers.net.
...   (13 gtld servers)
com.			86400	IN	DS	19718 13 2 8ACBB0CD28F41250A80A491389424D341522D946B0DA0C0291F2D3D7 71D7805A
com.			86400	IN	RRSIG	DS 8 1 86400 ...
;; Received 1171 bytes from 2001:500:a8::e#53(e.root-servers.net) in 9 ms

example.com.		172800	IN	NS	hera.ns.cloudflare.com.
example.com.		172800	IN	NS	elliott.ns.cloudflare.com.
example.com.		86400	IN	DS	2371 13 2 C988EC423E3880EB8DD8A46FE06CA230EE23F35B578D64E78B29C3E1 C83D245A
```

Real output, trimmed to the essential records. Three things to read here.

**The root answered, and it only returned `.com` servers.** It knows nothing about
`example.com` — only where to ask.

**You can see the query went over IPv6** — `2001:500:a8::e` is `e.root-servers.net`'s
IPv6 address. Root servers are among the most anycast-deployed services on the
internet, which is why the reply came back in 9 ms regardless of where the
resolver was.

**The `DS` record is DNSSEC.** The `.com` TLD publishes a cryptographic digest of
`example.com`'s signing key, and the root signs that. This chain of trust is what
lets a resolver verify that an answer was not tampered with. DNSSEC is deployed
for the root and most TLDs, but many registrars still do not sign individual
domains, so in practice much of the internet is unsigned.

### Why ICANN is not in the path

The source PDF for this module states DNS is "managed by ICANN". That is wrong in
a way worth correcting rather than repeating, because it misplaces the entire
architecture.

**ICANN coordinates, it does not operate.** Its role is policy and coordination:
it manages the root zone's administration, accredits and contracts with the
registry operators who run the TLDs, and delegates address space through the
Regional Internet Registries. It does not run recursive resolvers, and it does not
serve authoritative answers for your domain. Even the root servers are operated
by 12 separate organizations, not by ICANN — ICANN operates only `l`, one of the
13.

So your query never touches ICANN. It touches a resolver, then root, TLD, and
authoritative servers, each run by a different organization. This is delegation
working as designed.

## Caching

DNS is cached aggressively at every level, and this is what makes it work at
internet scale. Without caching, every lookup on earth would walk the full
hierarchy, and the root servers would be crushed.

| Cache | Where | Typical TTL |
|---|---|---|
| Browser | The application | Per its own policy |
| OS stub resolver | The kernel's resolver | Per `systemd-resolved` / nscd config |
| Recursive resolver | ISP or public resolver | Honours the record's TTL |

Confirm which resolver you are actually using:

```bash
dig example.com | grep -A1 'SERVER:'
```

```text
;; SERVER: 127.0.0.53#53(127.0.0.53) (UDP)
```

`127.0.0.53` is **not** a resolver on the internet — it is your own machine. This
is `systemd-resolved`'s stub, listening on a loopback address and forwarding to
your configured upstream. A consequence worth knowing: DNS queries pass through a
local process, so `dig` and your browser can get different answers if they are
configured differently, and you cannot see your real upstream by looking at
`/etc/resolv.conf`.

### TTL and propagation

Every record carries a **time to live**, the number of seconds it may be cached.
It is the single most important operational field in DNS, because it is a promise
about how long stale data may persist.

```bash
dig +noall +answer example.com
```

```text
example.com.		249	IN	A	104.20.23.154
example.com.		249	IN	A	172.66.147.243
```

That `249` is seconds remaining. It was 300 a moment ago and counted down.

**TTL is why DNS changes take time to propagate**, and it is a deliberate
trade-off. Lowering TTL before a planned change — waiting at least one old TTL,
then making the change, then raising TTL again — is standard practice, because a
TTL set too high means stale records are served long after you have fixed the
underlying problem.

The failure mode this creates is worth internalizing: **a record change is not
instant, and a cached negative answer is still an answer.** If a domain does not
resolve right after you create it, the likely cause is a cached NXDOMAIN, not a
missing record. Check what your resolver has cached before concluding the record
is absent.

## Record types

DNS is extensible: the type field identifies what a record contains, and new types
can be registered without changing the protocol.

| Type | Holds | Purpose |
|---|---|---|
| `A` | IPv4 address | The main one |
| `AAAA` | IPv6 address | The IPv6 equivalent |
| `CNAME` | Another name | Alias; cannot coexist with other records at that name |
| `MX` | Mail server + priority | Where mail for a domain goes |
| `NS` | Name server | Which servers are authoritative |
| `TXT` | Arbitrary text | SPF, DKIM, domain verification |
| `SOA` | Zone metadata | Start of authority; zone serial, admin contact |
| `PTR` | Name for an IP | Reverse lookup |

A `CNAME` cannot coexist with any other record type at the same name. This is not
an implementation quirk; it is in the protocol, and it surprises people
regularly. If a hostname needs both an alias and, say, an MX record, you must
invent a second hostname to hang the other record on.

### Reverse DNS

`A` maps name to address. `PTR` maps address back to name, in a separate tree
under `in-addr.arpa` (IPv4) or `ip6.arpa` (IPv6) — because the tree is keyed by
address in reverse order.

```bash
dig +short -x 203.0.113.10
```

Forward and reverse lookups are **not guaranteed to agree**. No rule requires a
`PTR` record to exist or to name the same host the `A` record points to. Mail
depends heavily on reverse DNS for reputation, and this mismatch is a real cause of
deliverable landing in spam folders.

## Practical experiment: observe it

Watch a resolution happen, including the cache:

```bash
# First query: cold
dig example.com

# Second, immediately: warm, and note the TTL has counted down
dig example.com
```

Compare the `;; Query time:` lines between the two. The first may take
milliseconds; the second is often `0 msec` because the resolver already has it.

Watch the query go over UDP rather than TCP:

```bash
dig +noall +stats example.com
```

```text
;; Query time: 0 msec
;; SERVER: 127.0.0.53#53(127.0.0.53) (UDP)
```

DNS normally uses UDP port 53. It falls back to TCP when the response does not fit
in a UDP datagram — zone transfers and some DNSSEC-signed responses exceed the
~512-byte limit that traditional DNS assumes, which is the origin of the EDNS0
`advertised UDP payload size` mechanism.

Check which servers are authoritative for a domain, and whether they are
reachable:

```bash
dig +noall +answer +authority example.com
```

```text
example.com.		249	IN	A	104.20.23.154
example.com.		249	IN	A	172.66.147.243
```

Without `+authority` you get only answers. Adding it also returns the NS records
that justify them — useful when you need to know *which* server is claiming
authority, which is the first question when a domain resolves to the wrong place.

### What to try next

- `dig example.com @1.1.1.1` then `dig example.com @8.8.8.8` then `dig
  example.com` with no server, and compare TTLs and timings. The first two bypass
  your ISP's resolver entirely.
- Set `+trace` against a domain you manage and read the delegation chain aloud.
- Compare `dig A example.com` with `dig AAAA example.com` on a dual-stack host.
  The AAAA answer is frequently empty on older setups, which silently degrades
  connections to IPv4-only paths.
- Watch a real connection open with `ss -tnp` while running `dig`, and confirm the
  DNS query happens before the TCP `SYN-SENT`.

## Common misconceptions

**"ICANN runs DNS."**
ICANN coordinates policy and the root zone; it operates one of thirteen root
server identities and runs no resolvers. The path from your machine is resolver →
root → TLD → authoritative, with ICANN in none of it.

**"The root server knows website addresses."**
It knows only which servers handle each TLD. It returns referrals until it is
asked about something outside its zone, which is never.

**"A DNS lookup returns one IP address."**
It can return several, and usually does. Multiple `A` or `AAAA` records are how
operators distribute load and provide failover. You must pick one and handle
failure — which is why clients try addresses in sequence.

**"DNS resolves once and that's it."**
Every connection re-resolves or reuses a cached answer, and the answer may have
expired and changed. This is why a long-lived process can start connecting to a
stale address after a migration.

**"DNS is only for websites."**
Every hostname you use goes through it: API endpoints, database hostnames, SMTP
servers, Kubernetes service discovery, and internal service names. Internal DNS
is often the most complex DNS an organisation operates, because it resolves names
that change far more often than public ones.

**"DNS is secure because it's on port 53."**
Plain DNS is unencrypted and unauthenticated: anyone on the path can read queries
and forge answers. DNSSEC authenticates but does not encrypt. DoH and DoT encrypt
the client-to-resolver leg, which is what prevents an observer on your local
network from reading or altering your lookups.

## Trade-offs

**Caching for speed against correctness for propagation.** Every cache entry is a
promise that recently-observed data is still true. This is why DNS changes are
eventually consistent and why TTL is an operational decision with a real cost
either way — low TTL means more load on the whole hierarchy, high TTL means
longer staleness after an incident.

**UDP for speed against size limits.** UDP keeps queries to one round trip and no
connection setup, at the cost of message size limits that require TCP fallback
and the EDNS0 machinery.

**A hierarchy for scalability against rigidity.** Delegation means no central
bottleneck, but adding a new TLD requires root zone changes and trust anchor
work, and the delegation chain means a misconfiguration anywhere breaks resolution
for everything beneath it.

**No authentication in the base protocol.** DNS was designed for a small,
trustworthy network. Adding security to it decades later (DNSSEC) required
retrofitting cryptographic signatures into a system where records are frequently
updated, and it is still not universally deployed. This is the clearest case in
modern infrastructure of a protocol whose assumptions outlived their context.

**The resolver is a trusted, opaque intermediary.** Because recursion hides the
hierarchy, you cannot see which server answered. This is convenient and also a
trust and observability problem: your resolver can return anything, and you will
believe it.

## Related concepts

- [All about the internet](./01-all-about-the-internet.md) — addressing, and why
  a name-to-address mapping is needed at all.
- [How data transfers](./02-how-data-transfers.md) — what happens to the query
  once it has an address.
- [TCP](./04-tcp.md) — DNS answers arrive over UDP, and DNSSEC responses can
  exceed UDP's size limit and fall back to TCP.
- [UDP](./05-udp.md) — why DNS is a natural fit for a connectionless transport.
- TLS, certificate validation, and name resolution — the security consequences
  of trusting a DNS answer.
- Service discovery in orchestration systems — internal DNS at scale.

## References

- RFC 1035 — Domain Names: Implementation and Specification. The original DNS
  specification, including the 13-root-server rationale.
  https://www.rfc-editor.org/rfc/rfc1035
- RFC 1034 — Domain Names: Concepts and Facilities. The hierarchy and delegation
  model. https://www.rfc-editor.org/rfc/rfc1034
- RFC 6891 — EDNS0. Extending UDP payload sizes beyond 512 bytes.
  https://www.rfc-editor.org/rfc/rfc6891
- RFC 4034 — Resource Record Types for DNSSEC (DS, RRSIG, and the rest).
  https://www.rfc-editor.org/rfc/rfc4034
- RFC 4035 — Protocol Modifications for DNSSEC. How validation works.
  https://www.rfc-editor.org/rfc/rfc4035
- RFC 7871 — Client Side DNS Caching. https://www.rfc-editor.org/rfc/rfc7871
- RFC 8499 — DNS Terminology. Precise definitions, including the recursive vs
  iterative distinction. https://www.rfc-editor.org/rfc/rfc8499
- IANA — Root Server list, with operator and anycast addresses.
  https://www.iana.org/domains/root/servers
- IANA — DNS Parameters and IANA registry of record types.
  https://www.iana.org/assignments/dns-parameters/dns-parameters.xhtml
- A. Tanenbaum, D. Wetherall, *Computer Networks*, 6th ed., Pearson, 2022,
  ch. 5.
- `dig(1)` and `resolv.conf(5)` — Linux man-pages.
  https://man7.org/linux/man-pages/man1/dig.1.html
