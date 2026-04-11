package runner

import (
	"testing"
)

func TestResolveTerminalFromConfig(t *testing.T) {
	term, err := ResolveTerminal("alacritty")
	if err != nil {
		t.Fatal(err)
	}
	if term != "alacritty" {
		t.Errorf("expected alacritty, got %s", term)
	}
}

func TestResolveTerminalFromEnv(t *testing.T) {
	t.Setenv("TERMINAL", "xterm")
	term, err := ResolveTerminal("")
	if err != nil {
		t.Fatal(err)
	}
	if term != "xterm" {
		t.Errorf("expected xterm from $TERMINAL, got %s", term)
	}
}

func TestTerminalArgsCommandIsLastArg(t *testing.T) {
	for _, terminal := range []string{"kitty", "alacritty", "foot", "ghostty"} {
		args := terminalArgs(terminal, "echo hi")
		if len(args) == 0 {
			t.Fatalf("%s: got empty args", terminal)
		}
		if args[0] != terminal {
			t.Errorf("%s: first arg should be terminal binary, got %q", terminal, args[0])
		}
		if args[len(args)-1] != "echo hi" {
			t.Errorf("%s: last arg should be command, got %q", terminal, args[len(args)-1])
		}
	}
}

func TestTerminalArgsWezterm(t *testing.T) {
	args := terminalArgs("wezterm", "echo hi")
	if len(args) < 2 || args[1] != "start" {
		t.Errorf("wezterm args should include 'start'; got %v", args)
	}
}

func TestTerminalArgsGnomeTerminal(t *testing.T) {
	args := terminalArgs("gnome-terminal", "echo hi")
	// gnome-terminal uses -- directly (no "start")
	found := false
	for _, a := range args {
		if a == "start" {
			found = true
		}
	}
	if found {
		t.Error("gnome-terminal should not include 'start' in args")
	}
}

func TestTerminalArgsFullPath(t *testing.T) {
	// When the terminal is specified as a full path, the binary name comparison
	// should still work correctly.
	args := terminalArgs("/usr/bin/wezterm", "echo hi")
	if len(args) < 2 || args[1] != "start" {
		t.Errorf("full-path wezterm should still include 'start'; got %v", args)
	}
}
