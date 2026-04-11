package runner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// terminalCandidates is checked in order when settings.terminal is not set.
var terminalCandidates = []string{
	"kitty",
	"alacritty",
	"wezterm",
	"ghostty",
	"foot",
	"gnome-terminal",
	"konsole",
	"xfce4-terminal",
	"xterm",
}

// ResolveTerminal returns the terminal emulator to use.
// It uses the configured value if set, then $TERMINAL, then auto-detects
// by scanning terminalCandidates for whatever is first found in PATH.
func ResolveTerminal(configured string) (string, error) {
	if configured != "" {
		return configured, nil
	}
	if t := os.Getenv("TERMINAL"); t != "" {
		return t, nil
	}
	for _, t := range terminalCandidates {
		if _, err := exec.LookPath(t); err == nil {
			return t, nil
		}
	}
	return "", fmt.Errorf("no terminal emulator found; set settings.terminal in your config")
}

// terminalArgs returns the argument list to launch cmd in the given terminal.
// Most terminals use `terminal -- cmd args...`; gnome-terminal uses `--`.
func terminalArgs(terminal, cmd string) []string {
	base := terminal
	// Strip any path prefix to get the binary name for comparison.
	if idx := strings.LastIndex(terminal, "/"); idx >= 0 {
		base = terminal[idx+1:]
	}
	switch base {
	case "wezterm":
		return []string{terminal, "start", "--", "/bin/sh", "-c", cmd}
	case "gnome-terminal", "xfce4-terminal":
		return []string{terminal, "--", "/bin/sh", "-c", cmd}
	default:
		// kitty, alacritty, foot, konsole, xterm, ghostty all accept:
		//   terminal -- /bin/sh -c "cmd"
		return []string{terminal, "--", "/bin/sh", "-c", cmd}
	}
}

// LaunchExternal spawns cmd in an external terminal emulator and returns
// immediately (the terminal runs independently). The terminal binary is
// resolved from configured, then $TERMINAL, then auto-detection.
func LaunchExternal(cmdStr, cwd, configured string, env map[string]string) error {
	terminal, err := ResolveTerminal(configured)
	if err != nil {
		return err
	}

	args := terminalArgs(terminal, cmdStr)
	c := exec.Command(args[0], args[1:]...)

	if cwd != "" {
		c.Dir = expandHome(cwd)
	}

	e := os.Environ()
	for k, v := range env {
		e = append(e, k+"="+v)
	}
	c.Env = e

	// Create a new session so the terminal emulator is fully detached from
	// burrow's process group. Without this, signals sent to burrow's group
	// (e.g. on window manager interactions) also kill the child.
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	return c.Start() // detached — do not wait
}
