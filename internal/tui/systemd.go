package tui

import (
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// servicesLoadedMsg carries the loaded service list.
type servicesLoadedMsg []serviceEntry

// serviceActionDoneMsg is returned when a start/stop/enable/disable completes.
type serviceActionDoneMsg struct {
	err    error
	output string
}

// serviceLogsMsg carries journalctl output for a service.
type serviceLogsMsg struct {
	name  string
	lines []string
}

// loadServicesCmd fetches all system and user services asynchronously.
func loadServicesCmd() tea.Cmd {
	return func() tea.Msg {
		sys := queryServices(false)
		usr := queryServices(true)
		return servicesLoadedMsg(append(sys, usr...))
	}
}

func queryServices(user bool) []serviceEntry {
	args := []string{"list-units", "--type=service", "--all", "--no-pager", "--plain", "--no-legend"}
	if user {
		args = append([]string{"--user"}, args...)
	}
	out, err := exec.Command("systemctl", args...).Output()
	if err != nil {
		return nil
	}

	args2 := []string{"list-unit-files", "--type=service", "--no-pager", "--plain", "--no-legend"}
	if user {
		args2 = append([]string{"--user"}, args2...)
	}
	out2, _ := exec.Command("systemctl", args2...).Output()
	enabledMap := parseEnabledStates(string(out2))

	return parseUnitList(string(out), enabledMap, user)
}

func parseEnabledStates(output string) map[string]string {
	m := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			m[fields[0]] = fields[1]
		}
	}
	return m
}

func parseUnitList(output string, enabledMap map[string]string, user bool) []serviceEntry {
	var services []serviceEntry
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		name := fields[0]
		if !strings.Contains(name, ".") {
			continue
		}
		active := fields[2]
		sub := fields[3]
		desc := strings.Join(fields[4:], " ")

		enabled := enabledMap[name]
		if enabled == "" {
			enabled = "-"
		}

		services = append(services, serviceEntry{
			name:        name,
			status:      active + "(" + sub + ")",
			enabled:     enabled,
			description: desc,
			user:        user,
		})
	}
	return services
}

// serviceActionCmd runs a systemctl action (start, stop, enable, disable) on a service.
func serviceActionCmd(svc serviceEntry, action string) tea.Cmd {
	return func() tea.Msg {
		args := []string{action, svc.name}
		if svc.user {
			args = append([]string{"--user"}, args...)
		}
		out, err := exec.Command("systemctl", args...).CombinedOutput()
		return serviceActionDoneMsg{err: err, output: strings.TrimSpace(string(out))}
	}
}

// serviceLogsCmd fetches the last 200 journalctl lines for a service.
func serviceLogsCmd(svc serviceEntry) tea.Cmd {
	return func() tea.Msg {
		args := []string{"-u", svc.name, "--no-pager", "-n", "200"}
		if svc.user {
			args = append([]string{"--user"}, args...)
		}
		out, err := exec.Command("journalctl", args...).Output()
		var lines []string
		if err == nil {
			text := strings.TrimSpace(string(out))
			if text != "" {
				lines = strings.Split(text, "\n")
			}
		}
		if len(lines) == 0 {
			lines = []string{"no logs available"}
		}
		return serviceLogsMsg{name: svc.name, lines: lines}
	}
}

// findUnitFilePath returns the filesystem path of the unit file for a service.
func findUnitFilePath(svc serviceEntry) string {
	args := []string{"show", "--property=FragmentPath", svc.name}
	if svc.user {
		args = append([]string{"--user"}, args...)
	}
	out, err := exec.Command("systemctl", args...).Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if val := strings.TrimPrefix(line, "FragmentPath="); val != line {
			return strings.TrimSpace(val)
		}
	}
	return ""
}
