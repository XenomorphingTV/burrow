package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/XenomorphingTV/burrow/internal/config"
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
	// Executor runs through /bin/sh -c so $VAR references in the command expand directly.
	task := config.Task{
		Cmd: "echo $BURROW_TEST_VAR",
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

func TestExecutorTimeout(t *testing.T) {
	task := config.Task{Cmd: "sleep 30", Timeout: 1}
	exec := NewExecutor("test-timeout", task, "manual", t.TempDir(), nil, nil)
	exec.Start()

	var exitCode int
	for line := range exec.LogCh() {
		if line.Done {
			exitCode = line.ExitCode
			break
		}
	}

	if exitCode == 0 {
		t.Error("expected non-zero exit code after timeout")
	}
}

func TestExecutorRetriesSucceedEventually(t *testing.T) {
	// Use a counter file so the command fails the first two attempts and
	// succeeds on the third.
	countFile := filepath.Join(t.TempDir(), "attempts")
	cmd := fmt.Sprintf(
		`c=$(cat %q 2>/dev/null || echo 0); c=$((c+1)); echo $c > %q; [ $c -ge 3 ] && echo "success" || exit 1`,
		countFile, countFile,
	)
	task := config.Task{Cmd: cmd, Retries: 3}
	exec := NewExecutor("test-retry-success", task, "manual", t.TempDir(), nil, nil)
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
		t.Errorf("expected exit 0 after retries, got %d; output: %v", exitCode, lines)
	}
	found := false
	for _, l := range lines {
		if strings.Contains(l, "success") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'success' in output, got: %v", lines)
	}
	// Two retry log lines should appear ([retry 1/3] and [retry 2/3]).
	retryLines := 0
	for _, l := range lines {
		if strings.HasPrefix(l, "[retry ") {
			retryLines++
		}
	}
	if retryLines != 2 {
		t.Errorf("expected 2 retry log lines, got %d; lines: %v", retryLines, lines)
	}
}

func TestExecutorRetriesExhausted(t *testing.T) {
	// A command that always fails; with Retries: 2 it should run 3 times total.
	task := config.Task{Cmd: "false", Retries: 2}
	exec := NewExecutor("test-retry-exhaust", task, "manual", t.TempDir(), nil, nil)
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

	if exitCode == 0 {
		t.Error("expected non-zero exit after all retries exhausted")
	}
	// Count "$ false" lines — one per attempt.
	attempts := 0
	for _, l := range lines {
		if l == "$ false" {
			attempts++
		}
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts (1 + 2 retries), got %d; lines: %v", attempts, lines)
	}
}

func TestExecutorRetriesLogsAccumulate(t *testing.T) {
	// All attempt output should end up in the single log file, not just the last attempt.
	logDir := t.TempDir()
	task := config.Task{Cmd: "echo attempt-output; exit 1", Retries: 1}
	exec := NewExecutor("test-retry-log", task, "manual", logDir, nil, nil)
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
		t.Fatal("expected a log file to be written")
	}
	data, err := os.ReadFile(filepath.Join(logDir, entries[0].Name()))
	if err != nil {
		t.Fatal(err)
	}
	// Two attempts → "attempt-output" should appear exactly twice as its own line
	// (the command echo line also contains the string, so we match whole lines).
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		if line == "attempt-output" {
			count++
		}
	}
	if count != 2 {
		t.Errorf("expected 'attempt-output' twice in log file (once per attempt), got %d; log:\n%s", count, data)
	}
}

func TestExecutorShellPipe(t *testing.T) {
	task := config.Task{Cmd: "echo hello-pipe | cat"}
	exec := NewExecutor("test-pipe", task, "manual", t.TempDir(), nil, nil)
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
		if strings.Contains(l, "hello-pipe") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'hello-pipe' in output via pipe, got: %v", lines)
	}
}
