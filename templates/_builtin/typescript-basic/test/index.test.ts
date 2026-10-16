import assert from "node:assert/strict";
import { test } from "node:test";
import { greet } from "../src/index.js";

test("greets the world by default", () => {
  assert.equal(greet(), "Hello, world! This is {{PROJECT_NAME}}.");
});

test("greets a name", () => {
  assert.match(greet("Ada"), /^Hello, Ada!/);
});
