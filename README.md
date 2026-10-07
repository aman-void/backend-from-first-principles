# Backend From First Principles

A technically rigorous reference for how backend systems actually work
underneath their abstractions.

Most backend material explains _how to use_ a tool. This repository explains
_what the tool is doing to your machine, your network, and your data_ while you
use it. The goal is a durable engineering reference: something you can come back
to in five years and still trust, and something an AI agent can load as reliable
context instead of guessing.

## The philosophy

Every note is built around one question:

> What is actually happening underneath the abstraction?

A note is not finished when it defines a term. It is finished when a reader can
explain the mechanism, observe it directly, and predict what will break. That
means going all the way down:

- **What** the thing is, and **why** it had to exist
- The **mental model** that makes the behavior predictable
- **How it works**, and what the operating system, network, runtime, or database
  is doing while it happens
- A **minimal runnable implementation** that exposes the mechanism
- A **practical experiment** the reader can run to see it themselves
- **Trade-offs**, **failure modes**, and **misconceptions**
- **Precise references** — RFCs and specifications first, never invented

Stopping at "TCP is a reliable transport protocol" is a failure. Explaining what
_reliable_ means as byte streams, sequence numbers, acknowledgements,
retransmission, ordering, and flow control is the actual work.

## Who it's for

1. Long-term personal engineering reference.
2. Developers learning backend engineering from the ground up.
3. Experienced engineers looking for a rigorous cross-check.
4. AI coding agents that need trustworthy backend context.

It is written to be **public and shareable**. Notes stand alone, terminology is
defined on first use, and claims are cited precisely enough to verify.

## Structure

The repository is a set of numbered modules. Each module is a directory, each
concept is a numbered note inside it, and the numbers encode learning order.

```text
backend-from-first-principles/
├── README.md              # this file — the map of the knowledge base
├── AGENTS.md              # conventions + the contract for AI agents
├── LICENSE                # CC BY 4.0 — prose and diagrams
├── LICENSE-CODE           # MIT — code examples
├── .gitignore             # excludes pdfs/, node_modules/, source material
├── pdfs/                  # source PDFs — gitignored, never committed
└── NN-module-name/        # a module, created on demand
    ├── README.md          # module index: scope, reading order, status
    ├── NN-concept.md      # one note per concept
    ├── examples/          # runnable reference code, when warranted
    │   ├── README.md
    │   └── build-tcp-server/
    │       ├── README.md
    │       ├── tcp-server-go/main.go
    │       └── tcp-server-typescript/main.ts
    └── assets/            # extracted figures — only when ASCII will not do
```

Modules are **created on demand**, when there is enough material to justify one.
No planned or empty modules are listed here.

| Module                            | Contents                                                                                     | Status      |
| --------------------------------- | -------------------------------------------------------------------------------------------- | ----------- |
| [01-networking](./01-networking/) | What the internet is, how data actually moves across it, and the protocols that make it work | In progress |

### 01-networking

1. [All about the internet](./01-networking/01-all-about-the-internet.md) — the
   orientation note: autonomous systems, packet switching, addressing, the
   end-to-end principle, and a first look at the network from the command line.
2. [How data transfers](./01-networking/02-how-data-transfers.md) — encapsulation,
   ARP, and why the IP address never changes while the MAC address is rewritten at
   every hop.
3. [DNS](./01-networking/03-dns.md) — the hierarchical lookup, watched from the
   root down with real `dig +trace` output.
4. [TCP](./01-networking/04-tcp.md) — reliability, flow control, and congestion
   control, and what each mechanism actually costs.
5. [UDP](./01-networking/05-udp.md) — the transport layer without any of that
   machinery, and when that is the better choice.

Runnable code lives in [`01-networking/examples/`](./01-networking/examples/):
[build-tcp-server](./01-networking/examples/build-tcp-server/) and
[build-udp-server](./01-networking/examples/build-udp-server/), each in Go and
TypeScript. They are reference implementations to copy into your own environment,
not a maintained project — no build file, no dependencies, single files only.

## How to read it

Start with a module's `README.md`. It gives the reading order, the
prerequisites for each note, and a start-here path. Notes declare their own
prerequisites in front matter, so you can walk the dependency graph in either
direction — forward from foundations, or backward from a topic you need today.

## Conventions

Full conventions live in **[AGENTS.md](./AGENTS.md)**, which is also the contract
for AI agents contributing here. The short version:

- One coherent concept per note. Never a grab-bag.
- **Diagrams are ASCII, in ` ```text ` fences.** Wire-format headers, packet
  layouts, and protocol sequences all belong in text — they render everywhere,
  diff as text, and stay correct when a field size turns out to be wrong.
- Code examples are reference implementations to copy into your own environment,
  written only when a concept requires them. Single file per language, standard
  library only. TypeScript examples carry a `tsconfig.json` for typechecking;
  there is no build step and `node_modules/` is never committed.
- Standard relative Markdown links. No wiki-links.
- Every non-obvious claim is sourced. Every reference is real.
- Nothing is called verified unless it actually was run.

## Accuracy over completeness

Never invent an RFC number, a default value, a benchmark, a flag, or a
citation. If something is uncertain, it gets investigated or flagged as
uncertain.

Reference priority: RFCs and standards → official specifications → official
documentation → academic papers → reputable technical books → established
engineering publications → community resources. A book is a starting point, not
an authority; where a book and a specification disagree, the specification wins
and the note says so.

Source PDFs and book scans live in `pdfs/`, which is gitignored and never
committed. Only original notes, recreated diagrams, selected extracted figures,
and citations go into git.

Source material is treated as a **starting point, not an authority.** The notes
in this repository were written from PDFs that were checked claim by claim
against the specifications, and several did not survive. Examples of
corrections that made it into the notes:

- DNS is not "managed by ICANN" — ICANN coordinates policy and delegates TLDs,
  and operates one of the thirteen root server identities.
- The TCP retransmission timeout has a 1-second floor per RFC 6298, not the
  "200ms to 3 seconds" the source stated.
- UDP's unit is a *datagram*, not a "segment".
- `3145728 ÷ 1460` leaves a remainder of 888 bytes, so the last segment is
  partial.

## Contributing

The repository branches by chapter: root files live on `main`, and each chapter
is developed on its own `chapter/NN-module-name` branch.

Read [AGENTS.md](./AGENTS.md) first — it defines the note structure, the front
matter schema, the branching model, the diagram policy, the completion bar a
note must clear before it can be marked stable, and the rules on verification
honesty. Two of them are worth repeating here:

- **Never claim something was tested when it wasn't.** Commands get run and
  their real output pasted. Code gets executed or marked `Unverified`. When a
  documented claim turns out to be wrong on running it, fix the claim and say
  what was actually measured.
- **Never commit or push without explicit permission.** Ask first, every time.

## License

- Prose and diagrams: [CC BY 4.0](./LICENSE)
- Code examples: [MIT](./LICENSE-CODE)

Figures extracted from copyrighted sources remain the property of their
copyright holders and are included with attribution for reference and study.
w
