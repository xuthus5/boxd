package main

import (
	"bytes"
	"os"
	"runtime"
	"testing"
)

func TestParseAndExecuteInternalHolder(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("network namespace holder requires Linux")
	}
	input, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	previous := os.Stdin
	os.Stdin = input
	t.Cleanup(func() {
		os.Stdin = previous
		if err := input.Close(); err != nil {
			t.Error(err)
		}
	})
	var stdout, stderr bytes.Buffer
	if code := parseAndExecute([]string{"--boxd-netns-holder"}, &stdout, &stderr); code != 0 {
		t.Fatalf("holder exit=%d stderr=%s", code, stderr.String())
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("holder wrote application startup output: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestParseAndExecuteInternalHolderRejectsArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := parseAndExecute([]string{"--boxd-netns-holder", "extra"}, &stdout, &stderr); code != 1 {
		t.Fatalf("invalid holder exit=%d", code)
	}
	if stdout.Len() != 0 || stderr.Len() == 0 {
		t.Fatal("invalid helper arguments must report an error without starting the app")
	}
}
