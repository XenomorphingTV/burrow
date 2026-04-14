package runner

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/XenomorphingTV/burrow/internal/config"
	"github.com/XenomorphingTV/burrow/internal/notify"
	"github.com/XenomorphingTV/burrow/internal/store"
)

// LogLine is a single line of output from a running task.
type LogLine struct {
	TaskName string
	Text     string
	IsErr    bool
	Done     bool // true = channel will close after this
	ExitCode int  // only valid when Done=true
}

// Executor manages a running task process.
type Executor struct {
	name          string
	task          config.Task
	trigger       string
	logDir        string
	store         store.Storer
	notifyDefault []string // global notify default from settings

	ch     chan LogLine
	cancel context.CancelFunc
}

// NewExecutor creates a new Executor for the given task.
// notifyDefault is the global settings.notify list; the task's own Notify
// field takes precedence over it.
func NewExecutor(name string, task config.Task, trigger, logDir string, st store.Storer, notifyDefault []string) *Executor {
	return &Executor{
		name:          name,
		task:          task,
		trigger:       trigger,
		logDir:        logDir,
		store:         st,
		notifyDefault: notifyDefault,
		ch:            make(chan LogLine, 256),
	}
}

// LogCh returns the read-only log channel.
func (e *Executor) LogCh() <-chan LogLine {
	return e.ch
}

// Start launches the task in a background goroutine.
func (e *Executor) Start() {
	var ctx context.Context
	var cancel context.CancelFunc

	if e.task.Timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(e.task.Timeout)*time.Second)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}

	e.cancel = cancel
	go e.run(ctx)
}

// Kill cancels the running task.
func (e *Executor) Kill() {
	if e.cancel != nil {
		e.cancel()
	}
}

// splitCommand splits a command string into fields, respecting single and double quotes.
func splitCommand(cmd string) ([]string, error) {
	var fields []string
	var current strings.Builder
	inSingle := false
	inDouble := false

	for i := 0; i < len(cmd); i++ {
		c := cmd[i]
		switch {
		case inSingle:
			if c == '\'' {
				inSingle = false
			} else {
				current.WriteByte(c)
			}
		case inDouble:
			if c == '"' {
				inDouble = false
			} else if c == '\\' && i+1 < len(cmd) {
				i++
				current.WriteByte(cmd[i])
			} else {
				current.WriteByte(c)
			}
		case c == '\'':
			inSingle = true
		case c == '"':
			inDouble = true
		case c == ' ' || c == '\t':
			if current.Len() > 0 {
				fields = append(fields, current.String())
				current.Reset()
			}
		default:
			current.WriteByte(c)
		}
	}
	if inSingle || inDouble {
		return nil, fmt.Errorf("unclosed quote in command: %q", cmd)
	}
	if current.Len() > 0 {
		fields = append(fields, current.String())
	}
	return fields, nil
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func (e *Executor) run(ctx context.Context) {
	startTime := time.Now()

	// Expand ~ in logDir
	logDir := expandHome(e.logDir)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		e.ch <- LogLine{TaskName: e.name, Text: fmt.Sprintf("[err] failed to create log dir: %v", err), IsErr: true}
	}

	// Build log file path (named by the run start time, shared across all attempts).
	timestamp := startTime.Format("20060102-150405")
	logFilePath := filepath.Join(logDir, fmt.Sprintf("%s-%s.log", e.name, timestamp))

	if strings.TrimSpace(e.task.Cmd) == "" {
		e.ch <- LogLine{TaskName: e.name, Text: "[err] empty command", IsErr: true, Done: true, ExitCode: 1}
		close(e.ch)
		return
	}

	// Collect all log lines across all attempts for file writing and history.
	var (
		logMu    sync.Mutex
		allLines []string
	)

	addLine := func(line string, isErr bool) {
		e.ch <- LogLine{TaskName: e.name, Text: line, IsErr: isErr}
		logMu.Lock()
		allLines = append(allLines, line)
		logMu.Unlock()
	}

	maxAttempts := 1 + e.task.Retries
	exitCode := 1

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Before each retry (not the first attempt), log and optionally wait.
		if attempt > 1 {
			delay := time.Duration(e.task.RetryDelay) * time.Second
			msg := fmt.Sprintf("[retry %d/%d]", attempt-1, e.task.Retries)
			if delay > 0 {
				msg += fmt.Sprintf(" retrying in %s...", delay)
			}
			addLine(msg, false)
			if delay > 0 {
				select {
				case <-ctx.Done():
					exitCode = 1
					goto finish
				case <-time.After(delay):
				}
			}
		}

		// Run through the shell so that $VAR expansion, pipes, redirects, and
		// other shell features work as users expect from a task runner.
		addLine("$ "+e.task.Cmd, false)
		cmd := exec.CommandContext(ctx, "/bin/sh", "-c", e.task.Cmd)
		if e.task.Cwd != "" {
			cmd.Dir = expandHome(e.task.Cwd)
		}
		env := os.Environ()
		for k, v := range e.task.Env {
			env = append(env, k+"="+v)
		}
		cmd.Env = env

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			addLine(fmt.Sprintf("[err] stdout pipe: %v", err), true)
			exitCode = 1
			continue
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			addLine(fmt.Sprintf("[err] stderr pipe: %v", err), true)
			exitCode = 1
			continue
		}
		if err := cmd.Start(); err != nil {
			addLine(fmt.Sprintf("[err] start: %v", err), true)
			exitCode = 1
			continue
		}

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				addLine(scanner.Text(), false)
			}
		}()
		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				addLine(scanner.Text(), true)
			}
		}()
		wg.Wait()

		exitCode = 0
		if err := cmd.Wait(); err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = 1
			}
		}

		if exitCode == 0 {
			break
		}
	}

finish:
	endTime := time.Now()
	durationMs := endTime.Sub(startTime).Milliseconds()

	logMu.Lock()
	lines := allLines
	logMu.Unlock()

	_ = os.WriteFile(logFilePath, []byte(strings.Join(lines, "\n")), 0644)

	// Keep the last 200 lines for the history record; full output is on disk.
	tail := lines
	if len(tail) > 200 {
		tail = tail[len(tail)-200:]
	}

	if e.store != nil {
		_ = e.store.Save(&store.RunRecord{
			ID:         fmt.Sprintf("%s-%s", e.name, timestamp),
			TaskName:   e.name,
			StartTime:  startTime,
			EndTime:    endTime,
			DurationMs: durationMs,
			ExitCode:   exitCode,
			Trigger:    e.trigger,
			LogTail:    tail,
		})
	}

	// Send desktop notification if subscribed.
	e.maybeNotify(exitCode, durationMs)

	e.ch <- LogLine{TaskName: e.name, Done: true, ExitCode: exitCode}
	close(e.ch)
}

func (e *Executor) maybeNotify(exitCode int, durationMs int64) {
	event := notify.EventSuccess
	if exitCode != 0 {
		event = notify.EventFailure
	}
	if !notify.ShouldNotify(event, e.task.Notify, e.notifyDefault) {
		return
	}

	dur := fmt.Sprintf("%.1fs", float64(durationMs)/1000)
	var body string
	if exitCode == 0 {
		body = fmt.Sprintf("Finished in %s [%s]", dur, e.trigger)
	} else {
		body = fmt.Sprintf("Failed (exit %d) after %s [%s]", exitCode, dur, e.trigger)
	}
	notify.Send(e.name, body, exitCode != 0)
}
