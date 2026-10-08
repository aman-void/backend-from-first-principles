# Build a TCP echo server

Runnable echo servers in Go and TypeScript, built to make one idea observable:
**the three-way handshake is not in your code, and TCP is a byte stream rather
than a message channel.**

Related note: [04-tcp.md](../../04-tcp.md)

## Layout

```text
build-tcp-server/
├── tcp-server-go/main.go            # net.Listen + goroutine per connection
├── tcp-server-typescript/main.ts    # net.createServer + one event loop
└── README.md                        # this file
```

Two languages, not because the logic is interesting twice, but because the
**concurrency model differs in a way that is itself the lesson.** Go gives each
connection its own goroutine and its own stack; Node runs every callback on a
single event loop. Same protocol, opposite architecture.

## Prerequisites

| Example | Requires | Verified on |
|---|---|---|
| Go | Go 1.22+ (standard library only, no dependencies) | go1.27.1 linux/amd64 |
| TypeScript | Node 20.6+ with `--experimental-strip-types`, Node 22.6+, or Bun 1.x | Node v26.7.0, Bun 1.4.2 |

The Go example needs nothing — no `go mod init`, no dependencies. It is one file
importing only `net`, `bufio`, `flag`, `fmt`, `log`, `strings`, and `time`.

The TypeScript example imports only `node:net`. Its `package.json` and
`tsconfig.json` exist **solely to typecheck**, with `typescript` and
`@types/node` as devDependencies.

```bash
cd tcp-server-typescript
bun install        # devDependencies, for typechecking only
bun run typecheck  # must exit 0
```

There is **no build step**. `node --experimental-strip-types` and `bun` both run
the TypeScript file directly.

## Running the Go version

```bash
cd tcp-server-go
go run main.go                   # listens on :8080, all interfaces
go run main.go 9000              # listens on :9000        (bare port argument)
go run main.go -port 9000        # same thing, explicit flag
go run main.go -host 127.0.0.1   # loopback only
```

Connect from another terminal:

```bash
nc 127.0.0.1 8080
```

```text
listening on [::]:8080
connect with:  nc 127.0.0.1 8080
stop with:     Ctrl-C

2026/10/07 23:03:14 accepted 127.0.0.1:43870 -> 127.0.0.1:8080
2026/10/07 23:03:14   read 6 bytes: "hello"
2026/10/07 23:03:14   read 6 bytes: "world"
2026/10/07 23:03:14 127.0.0.1:43870 disconnected (EOF)
```

Real output from a verified run. Type `hello`, press enter, type `world`, press
enter.

## Running the TypeScript version

```bash
cd tcp-server-typescript

# Node 20.6+ (or 22.6+ where the flag is no longer needed)
node --experimental-strip-types main.ts 8080
npm start -- 8080                  # same thing via the script

# Bun, no flag needed
bun main.ts 8080
bun main.ts --port 8080            # a bare port, --port N, and --port=N all work
```

Verified output under both runtimes:

```text
$ node --experimental-strip-types main.ts 8080
listening on :8080
connect with:  nc 127.0.0.1 8080
stop with:     Ctrl-C

accepted ::ffff:127.0.0.1:46126 -> ::ffff:127.0.0.1:8080
  data event: 11 bytes (total 11): "alpha\nbeta\n"

$ bun main.ts 8080
listening on :8080
accepted ::ffff:127.0.0.1:34258 -> ::ffff:127.0.0.1:8080
  data event: 12 bytes (total 12): "hello\nworld\n"
```

### Type safety

The `tsconfig.json` enables strictness deliberately rather than accepting
TypeScript's defaults:

| Flag | Why it is on |
|---|---|
| `strict` | The baseline: no implicit `any`, strict null checks |
| `noUncheckedIndexedAccess` | Makes `argv[i]` be `string \| undefined`, so index bounds must be checked instead of asserted |
| `exactOptionalPropertyTypes` | Distinguishes "absent" from "present and `undefined`" |
| `noImplicitReturns` | Every code path must return or throw |
| `noUnusedLocals` / `noUnusedParameters` | Catches dead code |
| `verbatimModuleSyntax` | Forces explicit `import type`, so types are not silently elided or emitted |
| `isolatedModules` | Guarantees the file compiles as a single unit, which is what Node's type-stripping does |

`noUncheckedIndexedAccess` is the one that catches real bugs, and it shaped the
argument parser:

```typescript
function argValue(flag: string, fallback: number): number {
  const spaced = argv.indexOf(flag);
  if (spaced !== -1) {
    const value = argv[spaced + 1];   // string | undefined, not string
    if (value !== undefined) return Number(value);
  }
  ...
}
```

Without that flag, `argv[spaced + 1]` would type as `string` and the bounds check
would be unnecessary as far as the compiler was concerned — the check would look
redundant and would eventually get removed. **The flag is what makes the runtime
check honest.**

```text
listening on :8080
connect with:  nc 127.0.0.1 8080
stop with:     Ctrl-C

accepted ::ffff:127.0.0.1:33962 -> ::ffff:127.0.0.1:8080
  data event: 12 bytes (total 12): "hello\nworld\n"
  ::ffff:127.0.0.1:33962 sent FIN (half-close) after 12 bytes
  ::ffff:127.0.0.1:33962 closed (error=false)
```

## What to observe

### 1. There is no SYN handling in your code

Neither example contains a single line that touches the handshake. `Accept()`
returning and the `'connection'` event firing both happen **after** the kernel
has already completed the three-way handshake.

Prove it: connect, and from a third terminal watch the socket transition.

```bash
ss -tn state syn-sent
```

`SYN-SENT` means the SYN is out and no SYN-ACK has come back — the handshake is
stalling. Your server code is irrelevant at that point; this is kernel work.

### 2. Chunking does not match your writes

This is the byte-stream lesson, and it is the one that breaks naive
protocol designs. The same 12 bytes arrived as **one** 12-byte `data` event when
piped through `nc`, but the same client doing two `write()` calls in the same
tick arrived as **two** 3-byte events:

```text
# piped at once — ONE event
printf 'hello\nworld\n' | nc -q1 127.0.0.1 8080
  data event: 12 bytes (total 12): "hello\nworld\n"

# two writes in one tick — TWO events
node -e 'const s=require("net").connect(8081,"127.0.0.1");s.on("data",d=>console.log(d.length));s.on("connect",()=>{s.write("aaa");s.write("bbb")})'
  data event: 3 bytes (total 3): "aaa"
  data event: 3 bytes (total 6): "bbb"
```

Same bytes, different chunking, decided by network timing you do not control.
**You cannot use a read or a `data` event as a message boundary.** If you need
framing, add a length prefix or a delimiter yourself.

### 3. The 4-tuple identifies the connection

The log line prints both addresses:

```text
accepted 127.0.0.1:43870 -> 127.0.0.1:8080
```

`43870` is the client's **ephemeral port**, assigned by the client kernel during
the handshake. `8080` is what the client asked for. That pair plus the two IP
addresses is the 4-tuple, and it is why a server can hold thousands of
simultaneous connections from the same client IP — each has a different source
port. Run two clients against the server and watch the remote ports differ.

### 4. Disconnection is not explicit code

There is no `handleDisconnect()` anywhere. `Accept` returning an error, a
zero-byte read returning `EOF`, and the `'close'` event are all the kernel
reporting a FIN. Notably, **`EOF` is a half-close**: the peer stopped writing but
can still read. The TypeScript version logs this explicitly.

### 5. Go blocks per connection, Node blocks never

Open the TypeScript version, uncomment the `socket.pause()` block in the
`'connection'` handler, and connect a second client. The second client is
accepted and served normally, because a paused socket does not block the event
loop.

The Go version achieves the same property differently — `go handle(conn)` returns
immediately and each connection blocks on its own goroutine. Both models let one
slow client fail to block others, which is the requirement; how they achieve it
is the difference worth internalizing.

## Things to change

- **Buffer your writes.** The echo writes back exactly what it read. Add a
  buffered writer and see how large writes change the read pattern.
- **Add framing.** Prefix each message with a 4-byte length and parse it in a
  loop, so the server can reassemble messages regardless of chunking. This is
  what a real protocol needs.
- **Add a connection limit.** Count active connections and refuse beyond a
  threshold. Watch what `Accept()` returns when the backlog fills.
- **Handle `CLOSE_WAIT` deliberately.** Add `defer conn.Close()` (already
  present), then comment out the close and send some data. You have just
  reproduced the most common TCP file-descriptor leak.
- **Half-close explicitly.** Keep reading after you have closed your write side,
  and observe that the peer can still deliver data.
- **Compare concurrency models.** Add a deliberate 5-second `sleep` (or
  `sleepSync`) in the handler and connect two clients at once. Go serves both
  concurrently; Node stalls the second until the first finishes unless you make
  the handler async.

## Related

- [04-tcp.md](../../04-tcp.md) — the note these examples demonstrate.
- [../build-udp-server/](../build-udp-server/) — the same servers without a
  handshake, which makes the contrast explicit.
