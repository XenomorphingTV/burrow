// Package notify sends desktop notifications via the OS-native mechanism.
//
// On Linux it shells out to notify-send, which speaks to the
// org.freedesktop.Notifications D-Bus interface. When running as a systemd
// user service DBUS_SESSION_BUS_ADDRESS is usually unset; the package
// automatically derives the socket path from XDG_RUNTIME_DIR (or /run/user/UID)
// so notifications work from daemon mode too.
//
// On macOS it uses osascript.
//
// Errors are intentionally soft — a missing notify-send binary or an
// unreachable D-Bus session is logged to stderr but never fatal.
package notify

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
)

// EventSuccess and EventFailure are the two notification trigger values.
const (
	EventSuccess = "success"
	EventFailure = "failure"
)

// ShouldNotify returns true when the given event matches the configured
// subscription list. taskNotify is checked first; if empty, globalNotify is
// used as the fallback. "always" in either list matches any event.
func ShouldNotify(event string, taskNotify, globalNotify []string) bool {
	list := taskNotify
	if len(list) == 0 {
		list = globalNotify
	}
	for _, v := range list {
		if v == event || v == "always" {
			return true
		}
	}
	return false
}

// Send dispatches a desktop notification. failure controls urgency (critical vs
// normal). Errors are non-fatal; the function logs to stderr and returns.
func Send(taskName, body string, failure bool) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = sendLinux(taskName, body, failure)
	case "darwin":
		err = sendDarwin(taskName, body)
	default:
		return
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "notify: %v\n", err)
	}
}

func sendLinux(title, body string, failure bool) error {
	urgency := "normal"
	if failure {
		urgency = "critical"
	}

	cmd := exec.Command("notify-send",
		"--urgency="+urgency,
		"--app-name=burrow",
		"burrow: "+title,
		body,
	)
	cmd.Env = sessionEnv()
	return cmd.Run()
}

// sessionEnv returns os.Environ() with DBUS_SESSION_BUS_ADDRESS set when it
// is absent, so notify-send works from a systemd user service.
func sessionEnv() []string {
	env := os.Environ()
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") != "" {
		return env
	}

	// Prefer XDG_RUNTIME_DIR if set, otherwise fall back to /run/user/UID.
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = "/run/user/" + strconv.Itoa(os.Getuid())
	}
	return append(env, "DBUS_SESSION_BUS_ADDRESS=unix:path="+runtimeDir+"/bus")
}

func sendDarwin(title, body string) error {
	script := fmt.Sprintf(`display notification %q with title %q`, body, "burrow: "+title)
	return exec.Command("osascript", "-e", script).Run()
}
