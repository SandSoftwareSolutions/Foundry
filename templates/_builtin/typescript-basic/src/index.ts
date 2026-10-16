import { fileURLToPath } from "node:url";

/** Returns a greeting. Kept separate from the command line so it is easy to test. */
export function greet(name: string = "world"): string {
  return `Hello, ${name}! This is {{PROJECT_NAME}}.`;
}

// Run only when this file is started directly (node dist/src/index.js [name]).
if (process.argv[1] === fileURLToPath(import.meta.url)) {
  console.log(greet(process.argv[2]));
}
