// A minimal TCP echo server, using only Node's built-in `net` module.
//
// What this demonstrates, and how it differs from the Go version:
//
//  1. The handshake is still NOT in your code. net.createServer() sets up the
//     listening socket; the kernel completes the three-way handshake before
//     your 'connection' callback fires. There is no SYN handler to write.
//
//  2. Node is single-threaded with an event loop. The 'connection' callback and
//     every socket callback run on the SAME thread, in order. That is why the
//     callbacks are non-blocking: a synchronous read inside one would stall
//     every other connection. Compare the Go version, where each connection gets
//     its own goroutine and its own stack.
//
//  3. Data arrives as Buffer chunks with no message boundaries, for exactly the
//     same reason as in TCP generally: it is a byte stream. 'data' here can be 1
//     byte or 64 KB, and two separate writes from the client may arrive as one
//     chunk or two, depending purely on timing. The header comment below has
//     three client variants that demonstrate the difference.
//
//  4. The server can be blocked deliberately with pause()/resume() to show that
//     one slow consumer does not block others — because the loop moves on.
//
// Run:
//
//	node main.ts                    # listens on :8080
//	node main.ts --port 9000
//
// Then in another terminal:
//
//	nc 127.0.0.1 8080
//	hello
//	world
//
// To see chunking that does NOT match your writes, compare these two runs:
//
//   # two writes in the same tick
//   node -e 'const s=require("net").connect(8080,"127.0.0.1");s.on("data",d=>console.log(d.length))'
//     → two 3-byte events
//
//   # two writes separated by a delay
//   node -e 'const s=require("net").connect(8080,"127.0.0.1");s.on("data",d=>console.log(d.length));s.on("connect",()=>{s.write("aaa");setTimeout(()=>s.write("bbb"),50)})'
//     → two 3-byte events
//
//   # two lines piped at once through nc
//   printf "hello\nworld\n" | nc 127.0.0.1 8080
//     → ONE 12-byte event
//
// Same bytes, different chunking, and it depends on timing you do not control.
// That is the entire reason you must not treat a 'data' event as a message.
import { createServer } from "node:net";

// Parse --port / -p without pulling in a dependency.
const argv = process.argv.slice(2);

// Supports "--port 9000" and "--port=9000".
// Note argv[i] is `string | undefined` under noUncheckedIndexedAccess, because
// an index expression is not proven to be in bounds. That is exactly why the
// bounds have to be checked rather than asserted.
function argValue(flag: string, fallback: number): number {
  const spaced = argv.indexOf(flag);
  if (spaced !== -1) {
    const value = argv[spaced + 1];
    if (value !== undefined) return Number(value);
  }

  const prefix = `${flag}=`;
  const inline = argv.find((arg) => arg.startsWith(prefix));
  if (inline !== undefined) {
    const value = inline.slice(prefix.length);
    if (value !== "") return Number(value);
  }

  return fallback;
}

// Also accept a bare positional port, so all of these work:
//
//	node main.ts 9000
//	node main.ts --port 9000
//	node main.ts --port=9000
//
// Without consuming flag.Args(), a positional argument is silently ignored and
// the server binds the default port anyway. That is a genuinely baffling
// failure: the process starts, prints a listening address, and your client
// still cannot reach it.
//
// process.argv[0] is the runtime and [1] is this script, so slice(2) is the
// caller's arguments.
function positionalPort(fallback: number): number {
  const args = argv.filter((arg) => !arg.startsWith("-"));
  // A bare number is the port. Anything else is left alone rather than guessed
  // at, because guessing wrong is worse than ignoring.
  for (const arg of args) {
    const n = Number(arg);
    if (Number.isInteger(n) && n > 0 && n <= 65535) return n;
  }
  return fallback;
}

// Explicit flags win; a bare positional port is the fallback.
const port = argValue("--port", argValue("-p", positionalPort(8080)));

// createServer() creates the LISTENING socket. Nothing is connected yet.
const server = createServer((socket) => {
  // The handshake is already complete. This callback is the first point at
  // which the connection exists as far as your code is concerned.
  //
  // The 4-tuple identity lives here: remotePort is the client's ephemeral port
  // chosen during the handshake, localPort is the port the client dialed.
  const remote = `${socket.remoteAddress}:${socket.remotePort}`;
  const local = `${socket.localAddress}:${socket.localPort}`;
  console.log(`accepted ${remote} -> ${local}`);

  // Per-connection state. In TCP there is no such thing as "the" read buffer,
  // but applications invent one — and they must decide their own framing.
  let totalBytes = 0;

  socket.on("data", (chunk: Buffer) => {
    totalBytes += chunk.length;

    // This log is the lesson. A client that writes twice may produce ONE data
    // event here, because TCP is a byte stream and segmentation is not visible
    // to you. Never assume one event == one message.
    console.log(
      `  data event: ${chunk.length} bytes (total ${totalBytes}): ` +
        `${JSON.stringify(chunk.toString("utf8"))}`,
    );

    // Echo. Note: no framing, no buffering decisions — write what arrived.
    // If this were a real protocol you would need to add a length prefix or a
    // delimiter here, because you cannot rely on chunk boundaries.
    socket.write(chunk);
  });

  socket.on("end", () => {
    // 'end' fires when the client sends FIN: it is done writing. This is a
    // HALF-close. The client may still be reading from us. Most echo servers
    // just end here, which implicitly ends our side too.
    console.log(`  ${remote} sent FIN (half-close) after ${totalBytes} bytes`);
  });

  socket.on("close", (hadError) => {
    console.log(`  ${remote} closed (error=${hadError})`);
  });

  socket.on("error", (err: Error) => {
    // ECONNRESET is what you get when the peer vanished — a RST, typically from
    // a middlebox, or from a client killed mid-transfer. You must handle it or
    // an unhandled 'error' event will crash the process.
    console.log(`  ${remote} error: ${err.message}`);
  });

  // ---- Deliberate demonstration -------------------------------------------
  // Uncomment pause() and connect a second client. The second client still
  // connects and is accepted, because a paused socket does not block the
  // event loop — it just stops this one socket from reading.
  //
  //   socket.pause()
  //   setTimeout(() => { console.log("  resuming"); socket.resume() }, 5000)
  //
  // This is the single most important thing to understand about Node's model.
});

server.on("error", (err: Error) => {
  console.error(`server error: ${err.message}`);
  process.exit(1);
});

// listen() binds the socket. The 'listening' event is when the address is
// assigned — listen() being called does not mean it succeeded.
server.listen(port, () => {
  const address = server.address();
  const bound = typeof address === "object" && address ? address.port : port;
  console.log(`listening on :${bound}`);
  console.log(`connect with:  nc 127.0.0.1 ${bound}`);
  console.log(`stop with:     Ctrl-C`);
  console.log();
});

process.on("SIGINT", () => {
  console.log("\nshutting down");
  // server.close() stops accepting NEW connections. Existing ones stay open.
  // Closing them requires tracking them — a real subtlety when shutting down.
  server.close(() => process.exit(0));
});
