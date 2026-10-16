import { fileURLToPath } from "node:url";

/** Returns a greeting. Kept separate from the command line so it is easy to test. */
export function greet(name = "world") {
  return `Hello, ${name}! This is {{PROJECT_NAME}}.`;
}

// Run only when this file is started directly (node src/index.js [name]).
if (process.argv[1] === fileURLToPath(import.meta.url)) {
  console.log(greet(process.argv[2]));
}
