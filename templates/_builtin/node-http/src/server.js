import { createServer } from "node:http";
import { fileURLToPath } from "node:url";

/** Builds the server without starting it, so tests can run it on a free port. */
export function buildServer() {
  return createServer((req, res) => {
    const send = (status, body) => {
      res.writeHead(status, { "Content-Type": "application/json" });
      res.end(JSON.stringify(body));
    };
    const { pathname } = new URL(req.url ?? "/", "http://localhost");
    if (pathname === "/healthz") return send(200, { status: "ok" });
    if (pathname === "/") return send(200, { message: "Hello from {{PROJECT_NAME}}" });
    return send(404, { error: "not found" });
  });
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const port = Number(process.env.PORT ?? 8080);
  const server = buildServer().listen(port, () => console.log(`{{PROJECT_NAME}} listening on :${port}`));
  process.on("SIGINT", () => server.close(() => process.exit(0)));
}
