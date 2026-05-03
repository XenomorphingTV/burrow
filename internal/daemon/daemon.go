package daemon

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/XenomorphingTV/burrow/internal/config"
	"github.com/XenomorphingTV/burrow/internal/runner"
	"github.com/XenomorphingTV/burrow/internal/store"
)

// SocketPath returns the Unix domain socket path.
func SocketPath() string {
	return filepath.Join(config.DefaultConfigDir(), "burrow.sock")
}

// PIDPath returns the PID file path.
func PIDPath() string {
	return filepath.Join(config.DefaultConfigDir(), "burrow.pid")
}

// IsRunning returns the daemon PID and whether it is currently running.
func IsRunning() (int, bool) {
	pid, err := readPID(PIDPath())
	if err != nil {
		return 0, false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return 0, false
	}
	// Signal 0 checks existence without killing.
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return 0, false
	}
	return pid, true
}

// daemon is the internal state of the running daemon process.
type daemon struct {
	mu        sync.RWMutex
	cfg       *config.Config
	cfgMtime  time.Time
	sched     *runner.Scheduler
	pool      *runner.Pool
	st        *store.Store
	listener  net.Listener
	heartbeat time.Time
	startTime time.Time

	watchMu      sync.Mutex
	watchCancels map[string]context.CancelFunc // taskName → cancel; one per active watcher
}

// Serve runs the daemon process. It blocks until SIGTERM or SIGINT is received.
// This is called by the hidden "daemon _serve" subcommand that the OS service
// file points at.
func Serve() error {
	cfgDir := config.DefaultConfigDir()
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	pidPath := PIDPath()
	if err := writePID(pidPath, os.Getpid()); err != nil {
		return fmt.Errorf("write pid: %w", err)
	}
	defer removePID(pidPath)

	st, err := store.Open(cfgDir)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer st.Close()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	maxParallel := cfg.Settings.MaxParallel
	if maxParallel <= 0 {
		maxParallel = 4
	}

	d := &daemon{
		cfg:          cfg,
		sched:        runner.NewScheduler(),
		pool:         runner.NewPool(maxParallel),
		st:           st,
		startTime:    time.Now(),
		heartbeat:    time.Now(),
		watchCancels: make(map[string]context.CancelFunc),
	}

	d.cfgMtime = globalConfigMtime()

	if err := d.registerSchedules(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: scheduler: %v\n", err)
	}
	disabled, _ := d.st.LoadDisabledSchedules()
	applyDisabledSchedules(d.sched, disabled)
	d.sched.Start()
	defer d.sched.Stop()

	d.startWatchers(disabled)
	defer d.stopWatchers()

	// Unix socket for IPC.
	sockPath := SocketPath()
	os.Remove(sockPath)
	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		return fmt.Errorf("listen on socket %s: %w", sockPath, err)
	}
	d.listener = ln
	defer ln.Close()
	defer os.Remove(sockPath)

	go d.acceptLoop(ln)
	go d.watchConfig()
	go d.heartbeatLoop()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

	for sig := range sigs {
		if sig == syscall.SIGHUP {
			if err := d.reload(); err != nil {
				fmt.Fprintf(os.Stderr, "warning: reload: %v\n", err)
			}
			continue
		}
		// SIGTERM or SIGINT — graceful shutdown.
		break
	}

	return nil
}

// registerSchedules adds all config schedules to d.sched.
// Caller must hold d.mu or be single-threaded.
func (d *daemon) registerSchedules() error {
	return d.sched.Register(d.cfg, d.onScheduledRun)
}

// applyDisabledSchedules enables or disables entries in sched to match disabled.
// It is safe to call before sched.Start().
func applyDisabledSchedules(sched *runner.Scheduler, disabled map[string]bool) {
	for _, e := range sched.AllSchedules() {
		if disabled[e.Name] {
			sched.Disable(e.Name)
		} else if !sched.IsEnabled(e.Name) {
			if err := sched.Enable(e.Name); err != nil {
				fmt.Fprintf(os.Stderr, "warning: re-enable schedule %q: %v\n", e.Name, err)
			}
		}
	}
}

// onScheduledRun is the callback given to the scheduler.
func (d *daemon) onScheduledRun(taskName, trigger string) {
	d.mu.RLock()
	_, ok := d.cfg.Tasks[taskName]
	d.mu.RUnlock()

	if !ok {
		fmt.Fprintf(os.Stderr, "scheduled run: task %q not found\n", taskName)
		return
	}

	go d.runPipeline(taskName, trigger)
}

// runPipeline resolves the dependency chain for target, runs each task in
// order, and fires on_failure/on_success hooks after the final task.
// It blocks until the whole pipeline finishes.
func (d *daemon) runPipeline(target, trigger string) {
	d.mu.RLock()
	cfg := d.cfg
	d.mu.RUnlock()

	ordered, err := runner.Resolve(target, cfg.Tasks)
	if err != nil || len(ordered) == 0 {
		fmt.Fprintf(os.Stderr, "daemon: resolve pipeline for %q: %v\n", target, err)
		return
	}

	// Run each task in dependency order; stop pipeline on first failure.
	exitCode := 0
	lastName := ordered[len(ordered)-1]
	for _, name := range ordered {
		exitCode = d.runTask(name, trigger)
		if exitCode != 0 {
			lastName = name
			break
		}
		// Only use "pipeline" trigger for intermediate deps; preserve trigger for target.
		trigger = "pipeline"
	}

	// Fire hooks on the task that finished (or failed).
	d.mu.RLock()
	task, ok := cfg.Tasks[lastName]
	logDir := cfg.Settings.LogDir
	notifyDefault := cfg.Settings.Notify
	d.mu.RUnlock()
	if !ok {
		return
	}

	if exitCode == 0 && task.OnSuccess != "" {
		if hookTask, ok := cfg.Tasks[task.OnSuccess]; ok {
			d.runTask(task.OnSuccess, "on_success")
			_ = hookTask
		} else {
			d.runShellHook(lastName+".on_success", task.OnSuccess, "on_success", logDir, notifyDefault)
		}
	} else if exitCode != 0 && task.OnFailure != "" {
		if hookTask, ok := cfg.Tasks[task.OnFailure]; ok {
			d.runTask(task.OnFailure, "on_failure")
			_ = hookTask
		} else {
			d.runShellHook(lastName+".on_failure", task.OnFailure, "on_failure", logDir, notifyDefault)
		}
	}
}

// runTask runs a single named task synchronously and returns its exit code.
func (d *daemon) runTask(name, trigger string) int {
	d.mu.RLock()
	task, ok := d.cfg.Tasks[name]
	logDir := d.cfg.Settings.LogDir
	notifyDefault := d.cfg.Settings.Notify
	d.mu.RUnlock()
	if !ok {
		return 1
	}

	d.pool.Acquire()
	exec := runner.NewExecutor(name, task, trigger, logDir, d.st, notifyDefault)
	exec.Start()
	exitCode := 0
	for line := range exec.LogCh() {
		if line.Done {
			exitCode = line.ExitCode
		}
	}
	d.pool.Release()
	return exitCode
}

// runShellHook runs a raw shell command (not a named task) as a hook.
func (d *daemon) runShellHook(execName, cmd, trigger, logDir string, notify []string) {
	d.pool.Acquire()
	task := config.Task{Cmd: cmd}
	exec := runner.NewExecutor(execName, task, trigger, logDir, d.st, notify)
	exec.Start()
	for range exec.LogCh() {
	}
	d.pool.Release()
}

// acceptLoop handles incoming IPC connections.
func (d *daemon) acceptLoop(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return // listener closed
		}
		go handleConn(conn, d)
	}
}

// watchConfig polls global config mtime and triggers a reload when it changes.
func (d *daemon) watchConfig() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		mtime := globalConfigMtime()
		d.mu.RLock()
		prev := d.cfgMtime
		d.mu.RUnlock()
		if !mtime.IsZero() && mtime.After(prev) {
			if err := d.reload(); err != nil {
				fmt.Fprintf(os.Stderr, "warning: config reload: %v\n", err)
			}
		}
	}
}

// reload reloads the config and re-registers all schedules.
func (d *daemon) reload() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Build and register new scheduler before swapping.
	newSched := runner.NewScheduler()
	if err := newSched.Register(cfg, d.onScheduledRun); err != nil {
		return fmt.Errorf("register schedules: %w", err)
	}
	// Restore disabled state so a reload doesn't re-enable user-disabled schedules.
	if disabled, err := d.st.LoadDisabledSchedules(); err == nil {
		applyDisabledSchedules(newSched, disabled)
	}

	d.mu.Lock()
	oldSched := d.sched
	d.cfg = cfg
	d.sched = newSched
	d.cfgMtime = globalConfigMtime()
	d.mu.Unlock()

	// Stop old scheduler after swap so no gap in coverage.
	oldSched.Stop()
	newSched.Start()

	// Restart file watchers for the new config.
	d.stopWatchers()
	reloadDisabled, _ := d.st.LoadDisabledSchedules()
	d.startWatchers(reloadDisabled)

	fmt.Fprintln(os.Stderr, "daemon: config reloaded")
	return nil
}

// heartbeatLoop updates d.heartbeat every 30 seconds.
func (d *daemon) heartbeatLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		d.mu.Lock()
		d.heartbeat = time.Now()
		d.mu.Unlock()
	}
}

// startWatchers launches a polling goroutine for every task that has Watch
// patterns and is not listed as disabled in the provided map.
// Caller must NOT hold d.watchMu.
func (d *daemon) startWatchers(disabled map[string]bool) {
	d.mu.RLock()
	cfg := d.cfg
	d.mu.RUnlock()

	d.watchMu.Lock()
	defer d.watchMu.Unlock()

	for taskName, task := range cfg.Tasks {
		if len(task.Watch) == 0 {
			continue
		}
		if disabled["watch:"+taskName] {
			continue
		}
		ctx, cancel := context.WithCancel(context.Background())
		d.watchCancels[taskName] = cancel
		go d.watchTask(ctx, taskName, task)
	}
}

// stopWatchers cancels all active file watcher goroutines and clears the map.
func (d *daemon) stopWatchers() {
	d.watchMu.Lock()
	defer d.watchMu.Unlock()
	for name, cancel := range d.watchCancels {
		cancel()
		delete(d.watchCancels, name)
	}
}

// applyDisabledWatchers starts or stops individual watchers to match disabled.
// Called when the TUI sends an updated disabled-schedules map over IPC.
func (d *daemon) applyDisabledWatchers(disabled map[string]bool) {
	d.mu.RLock()
	cfg := d.cfg
	d.mu.RUnlock()

	d.watchMu.Lock()
	defer d.watchMu.Unlock()

	for taskName, task := range cfg.Tasks {
		if len(task.Watch) == 0 {
			continue
		}
		key := "watch:" + taskName
		if disabled[key] {
			// Should be disabled — cancel if running.
			if cancel, ok := d.watchCancels[taskName]; ok {
				cancel()
				delete(d.watchCancels, taskName)
			}
		} else {
			// Should be enabled — start if not already running.
			if _, ok := d.watchCancels[taskName]; !ok {
				ctx, cancel := context.WithCancel(context.Background())
				d.watchCancels[taskName] = cancel
				go d.watchTask(ctx, taskName, task)
			}
		}
	}
}

// watchTask polls file mtimes for task and calls runWatchedTask when a change
// is detected. It blocks until ctx is cancelled.
func (d *daemon) watchTask(ctx context.Context, taskName string, task config.Task) {
	baseDir := task.Cwd
	if baseDir == "" {
		baseDir = "."
	}
	snap := runner.SnapshotFiles(task.Watch, baseDir)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			newSnap := runner.SnapshotFiles(task.Watch, baseDir)
			if !runner.SnapshotsMatch(snap, newSnap) {
				snap = newSnap
				d.runWatchedTask(ctx, taskName)
				if ctx.Err() != nil {
					return
				}
			}
		}
	}
}

// runWatchedTask runs the full pipeline for taskName synchronously (blocking
// until it finishes or ctx is cancelled). Running synchronously prevents
// stacking multiple triggered runs on top of each other.
func (d *daemon) runWatchedTask(ctx context.Context, taskName string) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		d.runPipeline(taskName, "watch")
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}
}

// globalConfigMtime returns the mtime of ~/.config/burrow/tasks.toml.
func globalConfigMtime() time.Time {
	path := filepath.Join(config.DefaultConfigDir(), "tasks.toml")
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

// writePID writes pid to path atomically.
func writePID(path string, pid int) error {
	return os.WriteFile(path, []byte(strconv.Itoa(pid)+"\n"), 0644)
}

// readPID reads a PID from path.
func readPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	// Trim whitespace/newline.
	s := string(data)
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r' || s[len(s)-1] == ' ') {
		s = s[:len(s)-1]
	}
	return strconv.Atoi(s)
}

// removePID removes the PID file if it contains our own PID.
func removePID(path string) {
	pid, err := readPID(path)
	if err != nil || pid != os.Getpid() {
		return
	}
	os.Remove(path)
}
