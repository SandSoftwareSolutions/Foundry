// {{PROJECT_NAME}} is a small command line program.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// version is set at build time: go build -ldflags "-X main.version=1.2.3".
var version = "dev"

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

// run holds the whole program, so tests can call it with their own arguments and buffers.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("{{PROJECT_NAME_KEBAB}}", flag.ContinueOnError)
	fs.SetOutput(stderr)
	name := fs.String("name", "world", "who to greet")
	showVersion := fs.Bool("version", false, "print the version and exit")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Fprintln(stdout, "{{PROJECT_NAME}}", version)
		return 0
	}
	fmt.Fprintf(stdout, "Hello, %s! This is {{PROJECT_NAME}}.\n", *name)
	return 0
}
