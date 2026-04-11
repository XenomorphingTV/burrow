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

	"github.com/xenomorphingtv/burrow/internal/config"
	"github.com/xenomorphingtv/burrow/internal/notify"
	"github.com/xenomorphingtv/burrow/internal/store"
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
	ctx, cancel := context.WithCancel(context.Background())
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

	// Build log file path
	timestamp := startTime.Format("20060102-150405")
	logFileName := fmt.Sprintf("%s-%s.log", e.name, timestamp)
	logFilePath := filepath.Join(logDir, logFileName)

	// Parse command using shell-aware splitting to handle quoted args
	fields, err := splitCommand(e.task.Cmd)
	if err != nil || len(fields) == 0 {
		e.ch <- LogLine{TaskName: e.name, Text: fmt.Sprintf("[err] invalid command: %v", err), IsErr: true, Done: true, ExitCode: 1}
		close(e.ch)
		return
	}

	// Expand ~ in each field so paths like ~/Scripts/foo.py work without a shell.
	for i, f := range fields {
		fields[i] = expandHome(f)
	}

	cmd := exec.CommandContext(ctx, fields[0], fields[1:]...)

	if e.task.Cwd != "" {
		cmd.Dir = expandHome(e.task.Cwd)
	}

	env := os.Environ()
	for k, v := range e.task.Env {
		env = append(env, k+"="+v)
	}
	cmd.Env = env

	e.ch <- LogLine{TaskName: e.name, Text: "$ " + e.task.Cmd}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		e.ch <- LogLine{TaskName: e.name, Text: fmt.Sprintf("[err] stdout pipe: %v", err), IsErr: true, Done: true, ExitCode: 1}
		close(e.ch)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		e.ch <- LogLine{TaskName: e.name, Text: fmt.Sprintf("[err] stderr pipe: %v", err), IsErr: true, Done: true, ExitCode: 1}
		close(e.ch)
		return
	}

	if err := cmd.Start(); err != nil {
		e.ch <- LogLine{TaskName: e.name, Text: fmt.Sprintf("[err] start: %v", err), IsErr: true, Done: true, ExitCode: 1}
		close(e.ch)
		return
	}

	// Collect all log lines for file writing and history
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

	exitCode := 0
	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	endTime := time.Now()
	durationMs := endTime.Sub(startTime).Milliseconds()

	logMu.Lock()
	lines := allLines
	logMu.Unlock()

	logContent := strings.Join(lines, "\n")
	_ = os.WriteFile(logFilePath, []byte(logContent), 0644)

	// Keep the last 200 lines for the history record; full output is on disk.
	tail := lines
	if len(tail) > 200 {
		tail = tail[len(tail)-200:]
	}

	id := fmt.Sprintf("%s-%s", e.name, timestamp)

	if e.store != nil {
		record := &store.RunRecord{
			ID:         id,
			TaskName:   e.name,
			StartTime:  startTime,
			EndTime:    endTime,
			DurationMs: durationMs,
			ExitCode:   exitCode,
			Trigger:    e.trigger,
			LogTail:    tail,
		}
		_ = e.store.Save(record)
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
