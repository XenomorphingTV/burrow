package runner

import (
	"fmt"
	"time"

	"github.com/XenomorphingTV/burrow/internal/config"
	"github.com/robfig/cron/v3"
)

// RunFunc is called when a scheduled task fires.
type RunFunc func(taskName, trigger string)

// registeredEntry holds everything needed to re-enable a schedule.
type registeredEntry struct {
	Spec     string
	TaskName string
	RunFn    RunFunc
}

// Scheduler wraps the robfig/cron v3 scheduler.
type Scheduler struct {
	c          *cron.Cron
	entries    map[string]cron.EntryID
	registered map[string]registeredEntry
}

// NewScheduler returns a Scheduler backed by robfig/cron.
func NewScheduler() *Scheduler {
	return &Scheduler{
		c:          cron.New(),
		entries:    make(map[string]cron.EntryID),
		registered: make(map[string]registeredEntry),
	}
}

// Register adds all schedules from the config to the cron scheduler.
func (s *Scheduler) Register(cfg *config.Config, runFn RunFunc) error {
	for name, sched := range cfg.Schedules {
		schedName := name
		taskName := sched.Task
		cronExpr := sched.Cron

		// Validate task exists
		if _, ok := cfg.Tasks[taskName]; !ok {
			return fmt.Errorf("schedule %q references unknown task %q", schedName, taskName)
		}

		s.registered[schedName] = registeredEntry{
			Spec:     cronExpr,
			TaskName: taskName,
			RunFn:    runFn,
		}

		capturedTaskName := taskName
		id, err := s.c.AddFunc(cronExpr, func() {
			runFn(capturedTaskName, "scheduled")
		})
		if err != nil {
			return fmt.Errorf("schedule %q invalid cron %q: %w", schedName, cronExpr, err)
		}
		s.entries[schedName] = id
	}
	return nil
}

// Disable removes a schedule from the cron engine but keeps it in registered.
func (s *Scheduler) Disable(name string) {
	if id, ok := s.entries[name]; ok {
		s.c.Remove(id)
		delete(s.entries, name)
	}
}

// Enable re-adds a schedule to the cron engine from registered data.
func (s *Scheduler) Enable(name string) error {
	e, ok := s.registered[name]
	if !ok {
		return fmt.Errorf("schedule %q not found in registered entries", name)
	}

	capturedTaskName := e.TaskName
	capturedRunFn := e.RunFn
	id, err := s.c.AddFunc(e.Spec, func() {
		capturedRunFn(capturedTaskName, "scheduled")
	})
	if err != nil {
		return fmt.Errorf("schedule %q invalid cron %q: %w", name, e.Spec, err)
	}
	s.entries[name] = id
	return nil
}

// IsEnabled returns true if the schedule is currently active in the cron engine.
func (s *Scheduler) IsEnabled(name string) bool {
	_, ok := s.entries[name]
	return ok
}

// NextRun returns the next scheduled run time for a schedule, or nil if disabled.
func (s *Scheduler) NextRun(name string) *time.Time {
	id, ok := s.entries[name]
	if !ok {
		return nil
	}
	entry := s.c.Entry(id)
	if entry.ID == 0 {
		return nil
	}
	t := entry.Next
	return &t
}

// UpdateSpec updates the cron expression for a registered schedule.
func (s *Scheduler) UpdateSpec(name, spec string) {
	if e, ok := s.registered[name]; ok {
		e.Spec = spec
		s.registered[name] = e
	}
}

// ScheduleEntry describes a registered schedule and its next run time.
type ScheduleEntry struct {
	Name     string
	TaskName string
	Spec     string
	Enabled  bool
	Next     *time.Time
}

// AllSchedules returns all registered schedules with their current state.
func (s *Scheduler) AllSchedules() []ScheduleEntry {
	entries := make([]ScheduleEntry, 0, len(s.registered))
	for name, reg := range s.registered {
		e := ScheduleEntry{
			Name:     name,
			TaskName: reg.TaskName,
			Spec:     reg.Spec,
			Enabled:  s.IsEnabled(name),
		}
		if t := s.NextRun(name); t != nil {
			e.Next = t
		}
		entries = append(entries, e)
	}
	return entries
}

// Start starts the cron engine.
func (s *Scheduler) Start() {
	s.c.Start()
}

// Stop halts the scheduler, waiting for running jobs to complete.
func (s *Scheduler) Stop() {
	ctx := s.c.Stop()
	<-ctx.Done()
}
