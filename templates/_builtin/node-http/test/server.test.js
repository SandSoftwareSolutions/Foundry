import assert from "node:assert/strict";
import { after, before, test } from "node:test";
import { buildServer } from "../src/server.js";

let server;
let base;

before(async () => {
  server = buildServer();
  await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
  base = `http://127.0.0.1:${server.address().port}`;
});

after(() => new Promise((resolve) => server.close(resolve)));

test("health endpoint", async () => {
  const res = await fetch(`${base}/healthz`);
  assert.equal(res.status, 200);
  assert.deepEqual(await res.json(), { status: "ok" });
});

test("root endpoint", async () => {
  const body = await (await fetch(base)).json();
  assert.match(body.message, /{{PROJECT_NAME}}/);
});

test("unknown path is a 404", async () => {
  assert.equal((await fetch(`${base}/nope`)).status, 404);
});
