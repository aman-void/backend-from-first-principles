# Examples — 01-networking

Runnable code for this module. These are **reference implementations to copy
into your own working environment**, not a maintained project. There is no root
build file, no dependency manifest, and no test suite; each example is a single
file using only its language's standard library.

Related note: each subdirectory links to the note it demonstrates.

| Example | Languages | Demonstrates |
|---|---|---|
| [build-tcp-server](./build-tcp-server/) | Go, TypeScript | Handshake is kernel work; TCP is a byte stream |
| [build-udp-server](./build-udp-server/) | Go, TypeScript | No handshake, boundaries preserved, silent loss |

## Why two languages per example

Not to duplicate the logic — the logic is deliberately boring. The reason is that
the **concurrency model differs in a way that is itself a lesson**:

- Go gives each connection its own goroutine and its own stack. Blocking is the
  default and is scoped to one connection.
- Node runs every callback on a single event loop. Blocking is a bug that stalls
  every connection at once.

Same protocol, opposite architecture. Reading both makes the difference
concrete in a way no prose can.

## Running them

Every Go example is a standalone single file with no dependencies:

```bash
cd build-tcp-server/tcp-server-go && go run main.go
```

Every TypeScript example imports only `node:*`. The `package.json`,
`tsconfig.json`, and `bun.lock` in those directories exist **solely to
typecheck** — `typescript` and `@types/node` as devDependencies, no build step:

```bash
cd build-tcp-server/tcp-server-typescript
bun install        # devDependencies only, for typechecking
bun run typecheck  # tsc --noEmit, must exit 0

bun main.ts 9000                                       # Bun runs it directly
node --experimental-strip-types main.ts 9000            # Node 20.6+
```

`node_modules/` is gitignored and never committed.

### Why the TypeScript files carry a tsconfig

TypeScript's default settings are not strict. The configs here turn on
`noUncheckedIndexedAccess`, `exactOptionalPropertyTypes`, `noImplicitReturns`,
`verbatimModuleSyntax`, and `isolatedModules`.

The one that matters most for teaching purposes is `noUncheckedIndexedAccess`:
it makes `argv[i]` type as `string | undefined`, which is *why* the argument
parsers check bounds instead of asserting them. Without the flag the compiler
would say the check was redundant, and it would eventually get removed.

## Layout

```text
examples/
├── README.md                              # this file
├── build-tcp-server/
│   ├── README.md
│   ├── tcp-server-go/main.go
│   └── tcp-server-typescript/
│       ├── main.ts
│       ├── package.json                   # typecheck only
│       ├── tsconfig.json                  # strictness flags
│       └── bun.lock
└── build-udp-server/
    ├── README.md
    ├── udp-server-go/main.go
    └── udp-server-typescript/
        ├── main.ts
        ├── package.json                   # typecheck only
        ├── tsconfig.json                  # strictness flags
        └── bun.lock
```

If you run `bun init` in one of these directories, delete the `CLAUDE.md` it
generates. A nested `CLAUDE.md` shadows repository-wide agent instructions, which
is actively harmful, and the stub `README.md` it writes would overwrite the
example's real documentation.

## Ports used

| Example | Default | Change with |
|---|---|---|
| TCP server | 8080 | `main.go 9000`, `-port 9000`, or `--port 9000` |
| UDP server | 9090 | `main.go 9000`, `-port 9000`, or `--port 9000` |

All four examples accept the same three forms. An earlier version documented a
bare positional port but never read `flag.Args()` / a positional scan, so the
argument was **silently ignored** and the server bound its default port anyway
while printing a perfectly normal listening banner. Verified fixed in all four.

`nc` needs `-u` for the UDP server, or it will speak TCP and appear broken.

## Verification status

**Go** — both examples executed with output captured verbatim in their READMEs.
Both are `gofmt`-clean and pass `go vet`.

**TypeScript** — both examples executed under Node v26.7.0 and Bun 1.4.2, and
`bun run typecheck` exits 0 under `strict` plus `noUncheckedIndexedAccess`,
`exactOptionalPropertyTypes`, `noImplicitReturns`, `noUnusedLocals`,
`noUnusedParameters`, `verbatimModuleSyntax`, and `isolatedModules`.

Five defects were found by review and running, and all five are fixed. They are
listed here because each one is a lesson in its own right:

1. **A bare positional port was silently ignored in all four examples.**
   The READMEs documented `go run main.go 9000`, but the code only read the
   `-addr` flag and never looked at `flag.Args()`, so the server bound its
   default port regardless — while printing an entirely normal listening banner.
   This is the worst class of bug in an example: it looks like it worked. Caught
   when a rerun failed with `address already in use` on port 8080. All four now
   accept a bare port, `-port N`, and `--port=N`.

2. **An absolute read deadline killed live connections.** The Go TCP example
   called `SetDeadline` once, before its read loop. `SetDeadline` is absolute,
   not relative, so the connection died after 60 seconds of total lifetime even
   if the client was actively using it. The deadline is now re-armed on every
   iteration, which makes it the idle timeout it was meant to be.

3. **Oversized datagrams were truncated with no error at all.** The Go UDP
   example read into a 1 KB buffer with no size check. `ReadFromUDP` does not
   error on an oversized datagram — it truncates and discards the rest silently,
   without ICMP. Measured: 4000 bytes sent, 2048 echoed back, sender told
   nothing. The example now uses a named `maxDatagram` constant and logs a
   warning when `n == len(buf)`, converting silent loss into a visible line.

4. **Chunking is timing-dependent, not predictable.** An earlier draft asserted
   that two writes in one tick always coalesce into a single `data` event.
   Measured behaviour was two separate 3-byte events, while 12 bytes piped
   through `nc` produced one 12-byte event. The TCP README now shows three client
   variants producing different chunking of identical bytes.

5. **`argv[i]` needs a bounds check.** With `noUncheckedIndexedAccess`, the
   original TypeScript parser failed to typecheck. The fix was to check the
   bound rather than assert it — which is the honest version of the code anyway,
   and is now used as the explanation of why that flag is worth having.

A sixth defect was caught by the compiler before the first release: the Go UDP
example shadowed the datagram length inside an `if` statement, so `go vet`
reported `declared and not used`. It now captures `WriteToUDP`'s return value.
