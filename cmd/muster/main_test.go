package main

import (
	"os"
	"testing"
)

func TestTerminalTypeAllowsTUI(t *testing.T) {
	for _, value := range []string{"dumb", " DUMB ", "DuMb"} {
		if terminalTypeAllowsTUI(value) {
			t.Fatalf("terminal type %q unexpectedly allows the TUI", value)
		}
	}
	for _, value := range []string{"", "xterm", "xterm-256color"} {
		if !terminalTypeAllowsTUI(value) {
			t.Fatalf("terminal type %q unexpectedly disables the TUI", value)
		}
	}
}

func TestIsTerminalRejectsNonTTYCharacterAndRegularFiles(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	device, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer device.Close()
	if isTerminal(device) {
		t.Fatal("/dev/null was treated as a terminal")
	}

	regular, err := os.CreateTemp(t.TempDir(), "regular")
	if err != nil {
		t.Fatal(err)
	}
	defer regular.Close()
	if isTerminal(regular) {
		t.Fatal("regular file was treated as a terminal")
	}
	if isTerminal(nil) {
		t.Fatal("nil file was treated as a terminal")
	}
}

func TestNoColorRequested(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if !noColorRequested() {
		t.Fatal("NO_COLOR=1 was ignored")
	}
	t.Setenv("NO_COLOR", "")
	if noColorRequested() {
		t.Fatal("empty NO_COLOR unexpectedly disabled color")
	}
}
