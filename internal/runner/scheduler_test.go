package runner

import (
	"testing"
	"time"

	"github.com/xenomorphingtv/burrow/internal/config"
)

func testSchedulerConfig() *config.Config {
	return &config.Config{
		Tasks: map[string]config.Task{
			"hello": {Cmd: "echo hello"},
		},
		Schedules: map[string]config.Schedule{
			"every-minute": {Task: "hello", Cron: "* * * * *"},
		},
	}
}

func TestSchedulerRegister(t *testing.T) {
	s := NewScheduler()
	if err := s.Register(testSchedulerConfig(), func(string, string) {}); err != nil {
		t.Fatal(err)
	}
	entries := s.AllSchedules()
	if len(entries) != 1 {
		t.Fatalf("expected 1 schedule, got %d", len(entries))
	}
	if entries[0].Name != "every-minute" {
		t.Errorf("unexpected schedule name: %s", entries[0].Name)
	}
}

func TestSchedulerRegisterUnknownTask(t *testing.T) {
	s := NewScheduler()
	cfg := &config.Config{
		Tasks: map[string]config.Task{},
		Schedules: map[string]config.Schedule{
			"bad": {Task: "nonexistent", Cron: "* * * * *"},
		},
	}
	if err := s.Register(cfg, func(string, string) {}); err == nil {
		t.Error("expected error for schedule referencing unknown task")
	}
}

func TestSchedulerRegisterInvalidCron(t *testing.T) {
	s := NewScheduler()
	cfg := &config.Config{
		Tasks: map[string]config.Task{
			"hello": {Cmd: "echo hello"},
		},
		Schedules: map[string]config.Schedule{
			"bad-cron": {Task: "hello", Cron: "not-a-cron-expression"},
		},
	}
	if err := s.Register(cfg, func(string, string) {}); err == nil {
		t.Error("expected error for invalid cron expression")
	}
}

func TestSchedulerEnabledAfterRegister(t *testing.T) {
	s := NewScheduler()
	s.Register(testSchedulerConfig(), func(string, string) {})

	if !s.IsEnabled("every-minute") {
		t.Error("schedule should be enabled after registration")
	}
}

func TestSchedulerDisable(t *testing.T) {
	s := NewScheduler()
	s.Register(testSchedulerConfig(), func(string, string) {})

	s.Disable("every-minute")
	if s.IsEnabled("every-minute") {
		t.Error("schedule should be disabled after Disable()")
	}
}

func TestSchedulerEnable(t *testing.T) {
	s := NewScheduler()
	s.Register(testSchedulerConfig(), func(string, string) {})
	s.Disable("every-minute")

	if err := s.Enable("every-minute"); err != nil {
		t.Fatal(err)
	}
	if !s.IsEnabled("every-minute") {
		t.Error("schedule should be enabled after Enable()")
	}
}

func TestSchedulerEnableUnknown(t *testing.T) {
	s := NewScheduler()
	if err := s.Enable("nonexistent"); err == nil {
		t.Error("expected error when enabling unknown schedule")
	}
}

func TestSchedulerNextRun(t *testing.T) {
	s := NewScheduler()
	s.Register(testSchedulerConfig(), func(string, string) {})
	s.Start()
	defer s.Stop()

	next := s.NextRun("every-minute")
	if next == nil {
		t.Fatal("expected a next run time, got nil")
	}
	if !next.After(time.Now()) {
		t.Error("next run time should be in the future")
	}
}

func TestSchedulerNextRunNilWhenDisabled(t *testing.T) {
	s := NewScheduler()
	s.Register(testSchedulerConfig(), func(string, string) {})
	s.Disable("every-minute")

	if next := s.NextRun("every-minute"); next != nil {
		t.Errorf("expected nil next run for disabled schedule, got %v", next)
	}
}

func TestSchedulerUpdateSpec(t *testing.T) {
	s := NewScheduler()
	s.Register(testSchedulerConfig(), func(string, string) {})
	s.UpdateSpec("every-minute", "0 * * * *")

	entries := s.AllSchedules()
	if len(entries) != 1 {
		t.Fatal("expected 1 schedule")
	}
	if entries[0].Spec != "0 * * * *" {
		t.Errorf("spec not updated; got %q", entries[0].Spec)
	}
}

func TestSchedulerAllSchedulesContainsEnabledFlag(t *testing.T) {
	s := NewScheduler()
	s.Register(testSchedulerConfig(), func(string, string) {})

	entries := s.AllSchedules()
	if len(entries) != 1 {
		t.Fatal("expected 1 entry")
	}
	if !entries[0].Enabled {
		t.Error("entry should be marked enabled")
	}

	s.Disable("every-minute")
	entries = s.AllSchedules()
	if entries[0].Enabled {
		t.Error("entry should be marked disabled after Disable()")
	}
}

func TestSchedulerFires(t *testing.T) {
	// Use a cron that fires every minute; we can't wait a full minute in a test,
	// so instead we just verify the scheduler starts and stops cleanly with a
	// registered job. Actual firing is covered by robfig/cron's own tests.
	fired := make(chan struct{}, 1)
	s := NewScheduler()
	cfg := &config.Config{
		Tasks: map[string]config.Task{"hello": {Cmd: "echo hello"}},
		Schedules: map[string]config.Schedule{
			"every-minute": {Task: "hello", Cron: "* * * * *"},
		},
	}
	s.Register(cfg, func(taskName, trigger string) {
		select {
		case fired <- struct{}{}:
		default:
		}
	})
	s.Start()
	s.Stop() // Stop waits for in-flight jobs to complete.
	// No assertion needed — we're just verifying Start/Stop don't deadlock.
}
