package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"text/template"

	"github.com/XenomorphingTV/burrow/internal/config"
)

// Install writes the OS service file without starting the daemon.
func Install(execPath string) error {
	switch runtime.GOOS {
	case "linux":
		return installSystemd(execPath)
	case "darwin":
		return installLaunchd(execPath)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// Uninstall stops and removes the OS service file.
func Uninstall() error {
	switch runtime.GOOS {
	case "linux":
		return uninstallSystemd()
	case "darwin":
		return uninstallLaunchd()
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// StartService starts the daemon. If the OS service unit is installed it is
// started through the service manager (preserving whatever WorkingDirectory was
// set at install time). Otherwise _serve is spawned directly as a detached
// background process in the caller's current working directory, so that the
// local burrow.toml in the project directory is picked up correctly.
func StartService(execPath string) error {
	switch runtime.GOOS {
	case "linux":
		if systemdUnitInstalled() {
			return systemctlUser("start", "burrow")
		}
		return spawnDirect(execPath)
	case "darwin":
		if launchdPlistInstalled() {
			return launchctl("load", launchdPlistPath())
		}
		return spawnDirect(execPath)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// StopService stops the daemon. If it is managed by the OS service manager it
// stops it there; otherwise it falls back to sending SIGTERM to the PID file.
func StopService() error {
	switch runtime.GOOS {
	case "linux":
		if systemdUnitActive() {
			return systemctlUser("stop", "burrow")
		}
		return killByPID()
	case "darwin":
		if launchdAgentLoaded() {
			return launchctl("unload", launchdPlistPath())
		}
		return killByPID()
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// killByPID sends SIGTERM to the PID stored in the PID file.
func killByPID() error {
	pid, err := readPID(PIDPath())
	if err != nil {
		return fmt.Errorf("read pid file: %w", err)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process %d: %w", pid, err)
	}
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("signal process %d: %w", pid, err)
	}
	return nil
}

// spawnDirect launches _serve as a detached background process in the caller's
// current working directory. Stdout and stderr are appended to the daemon log.
func spawnDirect(execPath string) error {
	logDir := filepath.Join(config.DefaultConfigDir(), "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}
	logPath := filepath.Join(logDir, "daemon.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open daemon log: %w", err)
	}
	defer logFile.Close()

	cmd := exec.Command(execPath, "daemon", "_serve")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("spawn daemon: %w", err)
	}
	// Do not wait — child is detached and runs independently.
	return nil
}

// systemdUnitInstalled returns true when the burrow.service unit file exists.
func systemdUnitInstalled() bool {
	path, err := systemdUnitPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// systemdUnitActive returns true when systemd considers burrow.service active.
func systemdUnitActive() bool {
	cmd := exec.Command("systemctl", "--user", "is-active", "--quiet", "burrow")
	return cmd.Run() == nil
}

// launchdPlistInstalled returns true when the burrow plist file exists on disk.
func launchdPlistInstalled() bool {
	_, err := os.Stat(launchdPlistPath())
	return err == nil
}

// launchdAgentLoaded returns true when launchd has the burrow agent loaded.
func launchdAgentLoaded() bool {
	cmd := exec.Command("launchctl", "list", "com.xenodium.burrow")
	return cmd.Run() == nil
}

const systemdUnit = `[Unit]
Description=Burrow task scheduler daemon
After=network.target

[Service]
ExecStart={{.ExecPath}} daemon _serve
WorkingDirectory={{.WorkingDir}}
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=default.target
`

func systemdUnitPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "systemd", "user", "burrow.service"), nil
}

func installSystemd(execPath string) error {
	path, err := systemdUnitPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create systemd dir: %w", err)
	}

	t := template.Must(template.New("unit").Parse(systemdUnit))
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create unit file: %w", err)
	}
	defer f.Close()

	cwd, err := os.Getwd()
	if err != nil {
		cwd = os.Getenv("HOME")
	}
	if err := t.Execute(f, struct {
		ExecPath   string
		WorkingDir string
	}{ExecPath: execPath, WorkingDir: cwd}); err != nil {
		return fmt.Errorf("write unit file: %w", err)
	}

	fmt.Printf("Installed: %s\n", path)

	// Reload daemon so systemd picks up the new unit.
	_ = systemctlUser("daemon-reload")
	return nil
}

func uninstallSystemd() error {
	_ = systemctlUser("stop", "burrow")
	_ = systemctlUser("disable", "burrow")

	path, err := systemdUnitPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove unit file: %w", err)
	}
	fmt.Printf("Removed: %s\n", path)
	_ = systemctlUser("daemon-reload")
	return nil
}

func systemctlUser(args ...string) error {
	cmd := exec.Command("systemctl", append([]string{"--user"}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

const launchdPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.xenodium.burrow</string>
	<key>ProgramArguments</key>
	<array>
		<string>{{.ExecPath}}</string>
		<string>daemon</string>
		<string>_serve</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>{{.LogPath}}</string>
	<key>StandardErrorPath</key>
	<string>{{.LogPath}}</string>
</dict>
</plist>
`

func launchdPlistPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Library", "LaunchAgents", "com.xenodium.burrow.plist")
}

func installLaunchd(execPath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	path := launchdPlistPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create LaunchAgents dir: %w", err)
	}

	logPath := filepath.Join(home, ".config", "burrow", "logs", "daemon.log")

	t := template.Must(template.New("plist").Parse(launchdPlist))
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create plist: %w", err)
	}
	defer f.Close()

	if err := t.Execute(f, struct {
		ExecPath string
		LogPath  string
	}{ExecPath: execPath, LogPath: logPath}); err != nil {
		return fmt.Errorf("write plist: %w", err)
	}

	fmt.Printf("Installed: %s\n", path)
	return nil
}

func uninstallLaunchd() error {
	path := launchdPlistPath()
	_ = launchctl("unload", path)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove plist: %w", err)
	}
	fmt.Printf("Removed: %s\n", path)
	return nil
}

func launchctl(args ...string) error {
	cmd := exec.Command("launchctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ServiceFilePath returns the OS-appropriate service file path (for display).
func ServiceFilePath() string {
	switch runtime.GOOS {
	case "linux":
		path, _ := systemdUnitPath()
		return path
	case "darwin":
		return launchdPlistPath()
	default:
		return "(unsupported OS)"
	}
}

// ServiceName returns a human-readable service manager description.
func ServiceName() string {
	switch runtime.GOOS {
	case "linux":
		return "systemd user service"
	case "darwin":
		return "launchd user agent"
	default:
		return "service"
	}
}

// expandHome expands a leading ~ in path p.
func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}
