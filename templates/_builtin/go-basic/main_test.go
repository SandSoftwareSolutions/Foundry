package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestGreeting(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"-name", "Ada"}, &out, &errOut); code != 0 {
		t.Fatalf("exit code %d, stderr: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Hello, Ada!") {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestVersion(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"-version"}, &out, &errOut); code != 0 || !strings.Contains(out.String(), version) {
		t.Fatalf("code %d, output %q", code, out.String())
	}
}

func TestBadFlag(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := run([]string{"-nope"}, &out, &errOut); code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
}
