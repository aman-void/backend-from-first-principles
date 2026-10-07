# AGENTS.md

Instructions for AI agents (and humans) contributing to this repository.

This file is the **contract**. If a contribution violates it, the change is
wrong regardless of how good the prose is. Read it before writing anything.

## Contents

1. What this repository is
2. Non-negotiable rules
3. Writing style
4. Repository layout
5. Note structure
6. Code examples
7. Diagrams
8. Terminology
9. Completion bar
10. Sources and figures
11. Repo roles and branching
12. Git hygiene

---

## 1. What this repository is

A technically rigorous backend engineering reference built from first
principles. The unit of value is not "documentation exists" but "a reader can
explain the mechanism underneath the abstraction after reading the note."

The organizing question for every note:

> What is actually happening underneath the abstraction the reader is using?

## 2. Non-negotiable rules

1. **Never invent technical facts.** No fabricated RFC numbers, flags, field
   sizes, header offsets, default values, syscalls, APIs, CLI flags, benchmark
   numbers, or citations. If you are unsure, investigate it or say you are
   unsure.
2. **Never invent references.** Every entry in a `## References` section must
   be a document that exists and that you have actually consulted. Verify the
   RFC number, the book's edition and chapter, the URL. A wrong citation is
   worse than no citation.
3. **Never claim something was tested when it was not.** If you ran it, say
   what you ran and paste the real output. If you did not, label it
   `Unverified` and say why. This applies to commands, code examples,
   benchmarks, and performance claims.
4. **Never renumber.** Filenames and their numeric prefixes are permanent.
5. **Never delete or rewrite correct material** for stylistic reasons. Fix
   errors and say what was wrong and why.
6. **Prose is original.** Do not paraphrase copyrighted text into near-copy.
   Summarize in your own words. Quote sparingly and attribute.
7. **Do not fabricate diagrams.** ASCII diagrams must depict the real
   mechanism. If you are unsure of the ordering or direction, leave it out.
8. **Do not pad.** A short, accurate note beats a long, padded one. Depth is
   proportional to the topic's actual complexity.

## 3. Writing style

Write like an experienced backend engineer teaching another engineer.

- Clear, precise, technical, practical, concise where possible, deep where
  necessary.
- Start with a concrete mechanism, not a definition of a noun.
- No marketing language, no motivational filler, no "In today's digital
  world...", no restating the reader's question back at them.
- Expand every acronym on first use. Do not assume the reader knows that
  `MTU` means maximum transmission unit.
- Prefer a concrete number or a real command over an adjective.

Bad:

> HTTP is a protocol used for communication on the internet.

Better:

> HTTP defines the semantics and wire format through which clients and servers
> exchange requests and responses. In a typical HTTP/1.1 deployment the
> application data is carried over a TCP byte stream, while HTTP/2 and HTTP/3
> change the underlying transport model.

Then explain each of those terms.

## 4. Repository layout

```
backend-from-first-principles/
├── README.md                 # map of the knowledge base (not a textbook)
├── AGENTS.md                 # this file
├── LICENSE                   # CC BY 4.0 — prose and diagrams
├── LICENSE-CODE              # MIT — code examples
├── .gitignore                # excludes pdfs/ and other source material
├── pdfs/                     # user's source PDFs — gitignored, never committed
└── NN-module-name/           # module, created on demand
    ├── README.md             # module index (required)
    ├── NN-concept.md         # one note per concept
    ├── examples/             # only when code is warranted
    │   ├── README.md
    │   └── verb-noun-task/
    │       ├── README.md
    │       └── task-lang/main.go | task-lang/main.ts
    └── assets/               # extracted figures, only when ASCII will not do
```

Examples are grouped by task, with a subdirectory per language:

```text
examples/
├── README.md
├── build-tcp-server/
│   ├── README.md
│   ├── tcp-server-go/main.go
│   └── tcp-server-typescript/main.ts
└── build-udp-server/
    ├── README.md
    ├── udp-server-go/main.go
    └── udp-server-typescript/main.ts
```

### Naming

| Thing           | Rule                            | Example                          |
| --------------- | ------------------------------- | -------------------------------- |
| Module dir      | two digits + kebab-case         | `01-networking`                  |
| Note file       | two digits + kebab-case + `.md` | `04-tcp.md`                      |
| Example dir     | verb-noun task name             | `build-tcp-server`               |
| Language subdir | `<task>-<language>`             | `tcp-server-go`, `udp-server-go` |
| Example file    | `main.<ext>`, one per language  | `main.go`, `main.ts`             |
| Asset file      | kebab-case, concept-named       | `tcp-timeline.png`               |

Notes are numbered contiguously within a module. The prefix defines learning
order. Never skip a number. Never reuse a number after deletion — leave a gap.

**Respect placeholder files.** If the user has created empty numbered files
(`03-dns.md`, `04-tcp.md`), they have chosen the numbering. Write into those
names; do not invent different numbers. If they disagree with the module README,
the placeholder wins — update the README to match.

Example dirs are named for the **task the reader performs**, not the concept,
because an example is an action: `build-tcp-server`, not `tcp-server`.

Assets are named after the **concept**, not the note number, so that renaming
or renumbering a note never orphans an image.

### Diagrams are ASCII, not images

**Do not commit raster diagrams of anything ASCII can represent.** Wire-format
headers, packet layouts, protocol sequences, and topology all belong in `text`
fences. They render identically on GitHub, in a terminal, and in any editor, they
diff line-by-line, and they stay editable when a field size turns out to be wrong.

Bit-layout headers render well as ASCII grids and are verified against the RFC.

Commit an image only when redrawing would genuinely lose information: a
photograph of hardware, a screenshot of a tool's actual output where the layout
itself is the lesson, or a hand-drawn figure that cannot be reproduced faithfully.

This was reversed once here. The TCP and UDP header diagrams were extracted from
a source PDF into `assets/` as PNGs, then deleted in favour of ASCII bit grids.
Do not repeat it.

### Creating a module

1. Take the next unused integer. Never reuse or reassign a number.
2. kebab-case name.
3. Write `README.md` with: scope, what deliberately lives elsewhere, notes in
   learning order with prerequisites, a start-here path, and status.
4. Do not add a module to the root `README.md` until the module exists.

Modules are created **on demand**. Do not speculatively create empty modules or
populate the root README with planned module names.

## 5. Note structure

Only four things are guaranteed. Everything else is chosen per topic.

```markdown
---
title: TCP
module: 01-networking
status: draft
prerequisites:
  - ./02-how-data-transfers.md
---

# TCP

<Opening paragraph: an accurate definition that would satisfy a competent
engineer. There is no "## Overview" heading — this paragraph is the overview.>

## Why does this exist?

<The problem that made this necessary.>

<Body sections chosen for this topic. Common candidates, none required:>

## Mental model

## How it works

## Under the hood

## Practical experiment

## Trade-offs

## Common misconceptions

## Related concepts

## References
```

Guaranteed: front matter, the opening definition paragraph,
`## Why does this exist?`, `## Related concepts`, `## References`.

`## References` and `## Related concepts` always come last.

### Front matter

```yaml
---
title: Human Readable Title
module: 01-networking
status: draft
prerequisites:
  - ./relative/path.md
---
```

- `title` — human-readable, not the filename.
- `module` — the module directory name.
- `status` — `draft` | `stable` | `needs-review`.
- `prerequisites` — relative paths to notes that should be read first.

There is deliberately **no `order` field**. The filename prefix already encodes
order; duplicating it guarantees rot.

`status` transitions:

- `draft` — written, not yet validated against the completion bar.
- `stable` — validated against the full completion bar in §9.
- `needs-review` — was correct, but the underlying technology or a cited source
  changed. Move it here rather than silently leaving it marked stable.

### Linking

Standard relative Markdown links only. No `[[wiki-links]]` — they do not render
as links on GitHub.

```markdown
[How data transfers](./02-how-data-transfers.md)
[TCP](./03-tcp.md)
[Other module](../../02-http-deep-dive/01-http-overview.md)
```

A **relative link to a note that does not exist yet is not allowed.** Link only
to files that exist. If a concept is relevant but unwritten, mention it as
plain text: "see `04-dns`, not yet written."

## 6. Code examples

Examples are **reference implementations to copy into your own working
environment**. They are not a maintained, integrated, CI-tested project.

- Code is written **when a concept requires it**, not by default.
- Not every concept gets an example. Notes-only modules are normal.
- Languages: Go and TypeScript, chosen per concept. Use `bash`, `sql`, or `c`
  when the concept is better shown that way.
- **One source file per example per language**, named `main.go` / `main.ts`.
  Nothing else. If an example needs a runtime dependency, it does not belong in
  this repository.
- Group by task, subdirectory by language: `build-tcp-server/tcp-server-go/main.go`.
- **TypeScript examples carry `package.json`, `tsconfig.json`, and `bun.lock`.**
  These exist **only to typecheck** — `typescript` and `@types/node` are
  devDependencies, and the example itself imports nothing but `node:*`. They are
  not a build system and there is no build step.
  - Enable strictness deliberately: `strict`, `noUncheckedIndexedAccess`,
    `exactOptionalPropertyTypes`, `noImplicitReturns`, `noUnusedLocals`,
    `noUnusedParameters`, `verbatimModuleSyntax`, `isolatedModules`.
  - `noUncheckedIndexedAccess` is the flag that catches real bugs — it makes
    `argv[i]` be `string | undefined`, which is why index bounds must be checked
    rather than asserted.
  - Verify with `bun run typecheck` (`tsc --noEmit`). It must exit 0.
  - `bun init` generates `CLAUDE.md`, a stub `README.md`, and `.gitignore`. Delete
    the `CLAUDE.md` and the stub README immediately — a nested `CLAUDE.md`
    shadowing repository-wide agent instructions is actively harmful, and the
    README belongs to the example task directory, one level up.
- **Go examples carry no `go.mod`.** A single stdlib-only `main.go` runs with
  `go run main.go` from any directory. To run `go vet`, copy it to a scratch
  directory and `go mod init` there — do not add a module file to the repository.
- `node_modules/` is gitignored. Never commit it.
- Prefer the standard library. Never introduce a framework to demonstrate
  something the standard library shows directly.
- Put the teaching in the **file header comment**. Explain what the program
  demonstrates, why each important API was chosen, and what to watch for —
  including the APIs that are deliberately *absent*. A reader should understand
  the mechanism before running it.
- Each example dir has a `README.md` covering: what it demonstrates,
  prerequisites with verified versions, how to run it standalone, **real captured
  output**, what to observe, and what to change to learn more.
- `examples/` itself gets a `README.md` listing the tasks and why each language is
  present.
- If an example has a genuine second angle worth showing in another language,
  say what the angle is. **A transliteration of the same logic is not worth
  writing.** The acceptable second angle is a different concurrency model, a
  different visibility into the mechanism, or a different failure mode.
- **Verify before shipping.** Run the example, capture the real output, and paste
  that output into the README. Typecheck TypeScript under `--strict` plus
  `noUncheckedIndexedAccess`; run `gofmt -l` and `go vet` on Go.
- If a documented claim turns out to be wrong when run, **fix the claim, not the
  expectation.** Note what was measured and what the real behaviour was. Never
  paste output you did not observe.
- Verification: state whether the example was executed. An unexecuted example is
  marked `Unverified`. Never imply otherwise.

## 7. Diagrams

**ASCII is the default**, and for wire formats it is not merely the default — it
is required. See §4 for the full rationale.

Rules:

- Fenced with ` ```text `. Never a bare fence — syntax highlighters mangle
  alignment.
- Maximum 80 columns wide for diagrams you author. **Verbatim command output is
  exempt**: trim by removing irrelevant rows, never by reflowing rows you keep.
- One glyph vocabulary across the repository:
  - `[actor]` for a participant or node
  - `-->` request / `←` response, or `--->` and `<---` for a clear pair
  - `|` for lifecycle boundaries and packet boundaries
  - `#` for a header field box, `.` for payload
  - `+-` and `|` for bit-grid borders in header layouts
- Show time flowing downward.
- Wire-format headers are drawn as a bit grid with a bit ruler, then verified
  field by field against the specification.
- Mermaid is permitted **only** where ASCII genuinely fails (dense multi-actor
  timing). Each use must be justified in the note; if you find yourself using it
  twice in a note, the ASCII is wrong.

Extracted figures from source PDFs live in the module's `assets/` and are
**always captioned with their source**:

```markdown
![Three-way handshake](./assets/tcp-three-way-handshake.png)

_Adapted from Stevens, TCP/IP Illustrated, Vol. 1, ch. 3._
```

**Recreate rather than extract, in every case.** Extract only when redrawing
would lose information that matters — a photograph, or a screenshot whose layout
is the lesson. Never extract a diagram that ASCII can express, however
authoritative the original. A Tanenbaum-style TCP header bitmap was extracted
from a source PDF in this repository and then deleted in favour of an ASCII bit
grid drawn from RFC 9293. That was the right call and it is the default.

## 8. Terminology

Define a term explicitly the first time it appears, and keep a dedicated
section for it if the concept deserves one.

Actively distinguish confusable pairs. This is a standing requirement, not
optional:

- HTTP vs TCP
- TCP vs IP
- Socket vs port
- Process vs thread
- Concurrency vs parallelism
- Latency vs bandwidth vs throughput
- Authentication vs authorization
- Encryption vs encoding
- Hashing vs encryption
- Flow control vs congestion control
- Replication vs sharding
- Primary key vs index
- Cache vs database

## 9. Completion bar

A note may only be `status: stable` when all of the following hold.

**Technical**

- [ ] Every non-obvious claim is sourced, or explicitly labeled as
      interpretation.
- [ ] Important distinctions are explained, not assumed.
- [ ] Every command shown was actually run; real output is pasted.
- [ ] Every example was executed, or carries an explicit `Unverified` marker.
- [ ] Every reference was actually consulted and is correctly cited.
- [ ] Every diagram is technically accurate against the spec.

**Structure**

- [ ] The note is in the correct module.
- [ ] Numbering is correct and contiguous.
- [ ] Every internal link resolves to a file that exists.
- [ ] Every asset path resolves.

**Teaching**

- [ ] It starts from fundamentals.
- [ ] It explains _why_, not only _what_.
- [ ] It explains what happens underneath.
- [ ] There is an experiment where one is appropriate.
- [ ] It links to prerequisites and to dependents.

**Maintainability**

- [ ] It follows this file's conventions.
- [ ] No duplication of an explanation that exists elsewhere — link instead.
- [ ] Filenames are predictable.

Report honestly. A note that passes 15 of 16 boxes is `draft`, not `stable`.

## 10. Sources and figures

Source PDFs, book scans, and articles are **stored on disk, not in git**. The
working drop folder is `pdfs/` at the repository root; it is gitignored, and so
are `*.pdf`, `*.epub`, `*.mobi`, `*.djvu`, `sources/`, and `_source/`.

The repository contains only original notes, redrawn diagrams, selected
extracted figures, and citations.

### Source material is a starting point, not an authority

The user supplies PDFs and screenshots as raw material. They are frequently
inexact. Before promoting any claim from them into a note:

1. **Verify every factual claim against the specification**, not just against the
   PDF. RFCs win. Note 01 of this module originally described DNS as "managed by
   ICANN" — the PDF said so, and it is wrong in a way that matters: ICANN
   coordinates the root zone and delegates TLDs, it does not run resolvers or
   authoritative servers.
2. **Check arithmetic.** The source PDF computed `3145728 ÷ 1460 ≈ 2155` and the
   real answer is 2155 with a remainder, so the segment count and the last
   segment's size both matter. Recompute.
3. **Discard illustrative values that look like facts.** A PDF's example ISN of
   `1000` or a host's IP of `104.21.16.1` is an example, not a measurement. Keep
   it only if it is clearly marked as illustrative.
4. **Fix silently-correct-but-wrong terminology.** A source calling UDP's unit a
   "segment" is using TCP's word for a different protocol's packet. Use the
   right term and, where the distinction is worth teaching, explain the
   difference.

### Attributing figures extracted from sources

- Prefer recreating as ASCII. Extract only when redrawing loses information —
  bit-layout diagrams and photographs are the usual cases.
- Put the file in the module's `assets/`, named after the **concept**, not the
  source or the note number.
- Always caption with the source, and say whether it was extracted or adapted:

  ```markdown
  ![TCP header format](./assets/tcp-header-format.png)

  *TCP header layout. Extracted from `pdfs/dns-tcp-udp.pdf`, p. 5 — originally
  a Tanenbaum-style diagram. The notes here are original; the figure is
  reproduced for reference.*
  ```

- When the original author of a figure is known and differs from the PDF's
  author, name both. Attribution must not credit the wrong person.
- Flatten extracted PNGs onto an opaque background before committing, so they do
  not render as black boxes on dark themes:

  ```bash
  magick input.png -background white -alpha remove -alpha off output.png
  ```

When working from a source document:

```
inspect document → locate relevant sections → extract concepts
  → inspect the figures inline, not just the text
  → verify claims against the RFC/spec, not only against the book
  → write original notes → recreate diagrams as ASCII
  → cite precisely (title, edition, chapter, page)
  → cross-link → validate against §9
```

A book is a starting point, not an authority. Where a book and a specification
disagree, the specification wins and the note says so.

### Reference formats

```
- RFC 9293 — Transmission Control Protocol (STD 7).
  https://www.rfc-editor.org/rfc/rfc9293
- J. Kurose, J. Ross, Computer Networking: A Top-Down Approach, 8th ed.,
  Pearson, 2021, ch. 3.
- socket(7) — Linux man-pages. https://man7.org/linux/man-pages/man7/socket.7.html
- PostgreSQL 16 Documentation, "CREATE INDEX".
  https://www.postgresql.org/docs/16/sql-createindex.html (retrieved 2026-01-15)
```

Source priority: RFCs and standards → official specifications → official
documentation → academic papers → reputable technical books → established
engineering publications → community resources.

A blog post or Stack Overflow answer may be cited, but must be marked
non-authoritative and its claim independently confirmed.

## 11. Repo roles and branching

### Branching model — the definitive version

**`main` holds root infrastructure and nothing else, forever. Every module lives
on its own permanent `chapter/NN-module-name` branch. Chapter branches are never
merged into `main` and never deleted.**

```
main                      # README.md, AGENTS.md, LICENSE, LICENSE-CODE, .gitignore
├── chapter/01-networking # permanent. notes + examples for module 01.
├── chapter/02-http-deep-dive
└── chapter/NN-module-name
```

Rules, with no exceptions:

- **Never merge a chapter branch into `main`.** `main` does not accumulate
  content.
- **Never delete a chapter branch.** It is the permanent home of its module.
- A chapter branch branches from `main` at the time the module is created.
  Branches never merge with each other.
- A note, diagram, example, or module README **never** lands on `main`. The only
  files that may change on `main` are the five root files.
- **Do not use `--delete-branch` when merging, and do not merge chapters at all.**
  If a user asks you to merge a chapter into `main`, treat that as a mistake and
  ask — it contradicts this section.
- Root files may need updating when a new chapter appears (the root `README.md`
  module table). That change goes on `main` **and** must be duplicated onto each
  chapter branch, or the chapter's copy will drift.

Because notes are not on `main`, **the root `README.md` must tell a reader how to
reach them**, and every module link in it is relative and therefore only resolves
on a chapter branch. Keep the "Branches" section at the top of the root README
for exactly this reason.

An earlier version of this file said chapter work "stays on that chapter's branch
until the user merges it," which implied chapters eventually merged into `main` and
directly contradicted the diagram above. That ambiguity caused a real mistake:
PR #1 was merged into `main` with `--delete-branch`, collapsing two branches into
one and putting module content on a branch that is supposed to hold none of it.
The wording above replaces it. If you are ever unsure which model is in force,
re-read this section — it is the authority.

| File                  | Role                                                                              |
| --------------------- | --------------------------------------------------------------------------------- |
| `README.md`           | The **map** of the knowledge base. Not a textbook. Lists only modules that exist, and explains the branch model. |
| `AGENTS.md`           | This file. Conventions and the validation contract.                               |
| `NN-module/README.md` | Index for one module: scope, note order, prerequisites, start-here path.          |
| `NN-concept.md`       | One concept.                                                                      |
| `NN-module/examples/` | Runnable reference code.                                                          |
| `NN-module/assets/`   | Figures.                                                                          |
| `LICENSE`             | CC BY 4.0 — prose and diagrams.                                                   |
| `LICENSE-CODE`        | MIT — code examples.                                                              |

**In-repo** output pasted into notes must be sanitized. Never commit real public
IP addresses, hostnames, MAC addresses, internal network names, usernames, or
credential material. Replace with the RFC 5737 documentation ranges
(`192.0.2.0/24`, `198.51.100.0/24`, `203.0.113.0/24`), `example.com`, or
`<your-ip>`. Sanitizing output is not falsifying it — but if you alter
measurements to make a point, say so explicitly.

## 12. Git hygiene

Focused, conventional commits. Never mix unrelated changes.

```
docs(networking): add TCP fundamentals
docs(http): explain request lifecycle
examples(tcp): add Go TCP echo server
docs(dns): explain recursive resolution
```

Never commit secrets, credentials, API keys, private documents, or source PDFs.

The branching model is defined **once**, in §11. Do not restate it here — two
slightly different copies of a branching rule is how this repository ended up
with two contradictory ones in the first place.

### Never commit or push without explicit permission

**Do not run `git commit` or `git push` unless the user has asked for it in the
current conversation.** Writing and editing files is expected; creating
permanent history and publishing it is not.

Specifically:

- Leaving work uncommitted is always acceptable. Report what changed instead.
- If you have staged changes, do not commit them just because the task is
  "finished". Finishing a task is not permission to commit.
- Never push, ever, without the user asking in that same request. Do not
  infer permission from an earlier approval of a different action.
- Never run `git commit --amend`, `git rebase`, `git reset --hard`, `git clean`,
  `git push --force`, or `git checkout .` / `git restore .` without asking.
  Destructive history operations are always opt-in.
- Never configure `commit.gpgsign`, hooks, or global git settings unprompted.
- If you believe committing is the right next step, say so and wait.

`git init`, `git add`, `git status`, `git diff`, `git log`, and other read-only
inspection are fine to run unprompted.
