// A minimal UDP echo server, using only Node's built-in `dgram` module.
//
// The deliberate contrast with the TCP server in this module. Every difference
// traces back to one fact: UDP has no handshake, no connection, no
// acknowledgements, and therefore no reliable delivery.
//
//  1. No 'connection'. bind() gives you a socket; there is no listen(), no
//     accept(), no 'connection' event, because there is nothing to accept.
//
//  2. One callback handles everything. Every incoming datagram triggers the same
//     'message' event, with (msg, rinfo). rinfo tells you the sender's address
//     and PORT — and the port is not stable, see note 4 below.
//
//  3. MESSAGE BOUNDARIES ARE PRESERVED. One datagram in, one datagram out, same
//     bytes, never merged, never split. This is the sharpest difference from TCP.
//
//  4. The source port is yours to choose. Unlike TCP, there is no handshake to
//     assign you an ephemeral port, so you set one explicitly with bind(), or
//     leave port 0 and let the OS pick. If you leave it 0, EVERY datagram will
//     appear to come from the same port. This is why DNS clients must bind a
//     random port themselves in order to match replies to queries.
//
//  5. NO FLOW CONTROL. If you are busy inside the 'message' handler when more
//     datagrams arrive, the kernel receive buffer fills and datagrams are
//     DROPPED. Node will not warn you, nothing is retransmitted, and the sender
//     never learns. Buffer, or lose.
//
// Run:
//
//	node main.ts                    # binds :9090
//	node main.ts --port 9000
//
// Then in another terminal:
//
//	nc -u 127.0.0.1 9090       # the -u is essential; without it you get nothing
//	hello
//
// One-shot:
//
//	echo -n 'hello' | nc -u -w1 127.0.0.1 9090
//
// To demonstrate the loss in point 5, send a burst and count what returns:
//
//	# 2000 datagrams, count the replies
//	for i in $(seq 2000); do echo -n "x" | nc -u -w1 127.0.0.1 9090; done | wc -l
//
// The count is often LESS than 2000. Nothing reports an error. That is UDP.
import { createSocket } from "node:dgram";

const argv = process.argv.slice(2);

// Supports both "--port 9000" and "--port=9000".
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
// Without consuming the positional arguments, `node main.ts 9000` would bind
// the default port anyway and print a listening address — a baffling way to
// lose a connection.
function positionalPort(fallback: number): number {
  const args = argv.filter((arg) => !arg.startsWith("-"));
  for (const arg of args) {
    const n = Number(arg);
    if (Number.isInteger(n) && n > 0 && n <= 65535) return n;
  }
  return fallback;
}

// Explicit flags win; a bare positional port is the fallback.
const port = argValue("--port", argValue("-p", positionalPort(9090)));

// 'udp4' restricts to IPv4. Use 'udp6' for IPv6-only, or 'udp' for both.
// Being explicit here avoids the dual-stack surprises you get with Node's
// default socket type.
const socket = createSocket("udp4");

let received = 0;
let bytesIn = 0;
let bytesOut = 0;

// THE central handler. Compare with the TCP server, which needed one callback
// per connection. Here a single callback serves every peer, because there are no
// connections to distinguish.
socket.on("message", (msg: Buffer, rinfo) => {
  received += 1;
  bytesIn += msg.length;

  // msg is EXACTLY one datagram. If the sender wrote 500 bytes you get 500
  // bytes here, whole, every time. TCP could not promise you that.
  console.log(
    `  ${msg.length} bytes from ${rinfo.address}:${rinfo.port}: ` +
      `${JSON.stringify(msg.toString("utf8"))}`,
  );

  // Echo back to rinfo — the address this specific datagram came from.
  // There is no session to remember: the server is entirely stateless, which is
  // why UDP scales trivially and is trivially spoofable.
  socket.send(msg, rinfo.port, rinfo.address, (err) => {
    if (err) {
      console.error(`  send failed: ${err.message}`);
      return;
    }
    bytesOut += msg.length;
  });
});

socket.on("error", (err: Error) => {
  // EADDRINUSE is the one you will hit most: something already holds the port.
  // Note that bind errors surface as an async 'error' event, not a throw, so
  // this handler is mandatory or the process crashes on startup.
  console.error(`socket error: ${err.message}`);
  process.exit(1);
});

// bind() is the UDP equivalent of listen()+accept() collapsed into one call.
// There is no backlog and nothing waits in it: datagrams that arrive before you
// bind are simply lost, because there was no socket to receive them.
socket.bind(port, () => {
  const addr = socket.address();
  console.log(`listening on :${addr.port} (udp4)`);
  console.log(`connect with:  nc -u 127.0.0.1 ${addr.port}`);
  console.log(`stop with:     Ctrl-C`);
  console.log();
});

// Running totals. Compare against what you sent: the gap is real loss, and UDP
// will never mention it.
const timer = setInterval(() => {
  if (received > 0) {
    console.log(`[dgrams=${received}  bytesIn=${bytesIn}  echoed=${bytesOut}]`);
  }
}, 1000);
timer.unref();

process.on("SIGINT", () => {
  console.log("\nshutting down");
  // For UDP, close() is immediate and total. There are no established
  // connections to drain — the TCP server's shutdown dance has no analogue here.
  socket.close(() => process.exit(0));
});
