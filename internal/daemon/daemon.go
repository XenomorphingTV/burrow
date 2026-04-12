package daemon

import (
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
		cfg:       cfg,
		sched:     runner.NewScheduler(),
		pool:      runner.NewPool(maxParallel),
		st:        st,
		startTime: time.Now(),
		heartbeat: time.Now(),
	}

	d.cfgMtime = globalConfigMtime()

	if err := d.registerSchedules(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: scheduler: %v\n", err)
	}
	if disabled, err := d.st.LoadDisabledSchedules(); err == nil {
		applyDisabledSchedules(d.sched, disabled)
	}
	d.sched.Start()
	defer d.sched.Stop()

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
	task, ok := d.cfg.Tasks[taskName]
	logDir := d.cfg.Settings.LogDir
	notifyDefault := d.cfg.Settings.Notify
	d.mu.RUnlock()

	if !ok {
		fmt.Fprintf(os.Stderr, "scheduled run: task %q not found\n", taskName)
		return
	}

	d.pool.Acquire()
	exec := runner.NewExecutor(taskName, task, trigger, logDir, d.st, notifyDefault)
	exec.Start()
	go func() {
		defer d.pool.Release()
		for range exec.LogCh() {
		}
	}()
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
