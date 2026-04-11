package runner

import (
	"os"
	"strings"
	"testing"

	"github.com/xenomorphingtv/burrow/internal/config"
)

// splitCommand tests

func TestSplitCommandBasic(t *testing.T) {
	fields, err := splitCommand("echo hello world")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"echo", "hello", "world"}
	if len(fields) != len(want) {
		t.Fatalf("expected %v, got %v", want, fields)
	}
	for i, f := range fields {
		if f != want[i] {
			t.Errorf("field[%d]: got %q, want %q", i, f, want[i])
		}
	}
}

func TestSplitCommandDoubleQuotes(t *testing.T) {
	fields, err := splitCommand(`echo "hello world"`)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 2 || fields[1] != "hello world" {
		t.Errorf("expected [echo, \"hello world\"], got %v", fields)
	}
}

func TestSplitCommandSingleQuotes(t *testing.T) {
	fields, err := splitCommand("echo 'hello world'")
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 2 || fields[1] != "hello world" {
		t.Errorf("expected [echo, hello world], got %v", fields)
	}
}

func TestSplitCommandEscapeInDoubleQuotes(t *testing.T) {
	fields, err := splitCommand(`echo "say \"hi\""`)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 2 || fields[1] != `say "hi"` {
		t.Errorf("unexpected result: %v", fields)
	}
}

func TestSplitCommandLeadingTrailingSpaces(t *testing.T) {
	fields, err := splitCommand("  echo  hello  ")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"echo", "hello"}
	if len(fields) != len(want) {
		t.Fatalf("expected %v, got %v", want, fields)
	}
}

func TestSplitCommandEmpty(t *testing.T) {
	fields, err := splitCommand("")
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 0 {
		t.Errorf("expected empty slice for empty input, got %v", fields)
	}
}

func TestSplitCommandUnclosedDoubleQuote(t *testing.T) {
	_, err := splitCommand(`echo "unclosed`)
	if err == nil {
		t.Error("expected error for unclosed double quote")
	}
}

func TestSplitCommandUnclosedSingleQuote(t *testing.T) {
	_, err := splitCommand("echo 'unclosed")
	if err == nil {
		t.Error("expected error for unclosed single quote")
	}
}

func TestSplitCommandMixedQuotes(t *testing.T) {
	fields, err := splitCommand(`node -e 'console.log("hi")'`)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 3 {
		t.Fatalf("expected 3 fields, got %v", fields)
	}
	if fields[2] != `console.log("hi")` {
		t.Errorf("unexpected third field: %q", fields[2])
	}
}

// executor integration tests — spawn real processes

func TestExecutorRunsCommand(t *testing.T) {
	task := config.Task{Cmd: "echo burrow-test-output"}
	exec := NewExecutor("test-echo", task, "manual", t.TempDir(), nil, nil)
	exec.Start()

	var lines []string
	var exitCode int
	for line := range exec.LogCh() {
		if line.Done {
			exitCode = line.ExitCode
			break
		}
		lines = append(lines, line.Text)
	}

	if exitCode != 0 {
		t.Errorf("expected exit 0, got %d", exitCode)
	}

	found := false
	for _, l := range lines {
		if strings.Contains(l, "burrow-test-output") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'burrow-test-output' in output, got: %v", lines)
	}
}

func TestExecutorFailingCommand(t *testing.T) {
	task := config.Task{Cmd: "false"}
	exec := NewExecutor("test-fail", task, "manual", t.TempDir(), nil, nil)
	exec.Start()

	var exitCode int
	for line := range exec.LogCh() {
		if line.Done {
			exitCode = line.ExitCode
			break
		}
	}

	if exitCode == 0 {
		t.Error("expected non-zero exit code for 'false'")
	}
}

func TestExecutorKill(t *testing.T) {
	task := config.Task{Cmd: "sleep 60"}
	exec := NewExecutor("test-kill", task, "manual", t.TempDir(), nil, nil)
	exec.Start()
	exec.Kill()

	for line := range exec.LogCh() {
		if line.Done {
			if line.ExitCode == 0 {
				t.Error("expected non-zero exit code after kill")
			}
			return
		}
	}
}

func TestExecutorWritesLogFile(t *testing.T) {
	logDir := t.TempDir()
	task := config.Task{Cmd: "echo log-file-test"}
	exec := NewExecutor("test-logfile", task, "manual", logDir, nil, nil)
	exec.Start()

	for line := range exec.LogCh() {
		if line.Done {
			break
		}
	}

	entries, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Error("expected a log file to be written in logDir")
	}
}

func TestExecutorInvalidCommand(t *testing.T) {
	task := config.Task{Cmd: `echo "unclosed`}
	exec := NewExecutor("test-invalid", task, "manual", t.TempDir(), nil, nil)
	exec.Start()

	for line := range exec.LogCh() {
		if line.Done {
			if line.ExitCode == 0 {
				t.Error("expected non-zero exit for invalid command")
			}
			return
		}
	}
}

func TestExecutorEnvVars(t *testing.T) {
	task := config.Task{
		Cmd: "sh -c 'echo $BURROW_TEST_VAR'",
		Env: map[string]string{"BURROW_TEST_VAR": "hello-from-env"},
	}
	exec := NewExecutor("test-env", task, "manual", t.TempDir(), nil, nil)
	exec.Start()

	var lines []string
	for line := range exec.LogCh() {
		if line.Done {
			break
		}
		lines = append(lines, line.Text)
	}

	found := false
	for _, l := range lines {
		if strings.Contains(l, "hello-from-env") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected env var in output, got: %v", lines)
	}
}
