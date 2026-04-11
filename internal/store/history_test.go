package store

import (
	"fmt"
	"testing"
	"time"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()
	st, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func makeRecord(id, taskName string, exitCode int, start time.Time) *RunRecord {
	return &RunRecord{
		ID:         id,
		TaskName:   taskName,
		StartTime:  start,
		EndTime:    start.Add(time.Second),
		DurationMs: 1000,
		ExitCode:   exitCode,
		Trigger:    "manual",
		LogTail:    []string{"output line"},
	}
}

func TestSaveAndRecent(t *testing.T) {
	st := openTestStore(t)
	now := time.Now()

	r1 := makeRecord("r1", "task-a", 0, now.Add(-2*time.Minute))
	r2 := makeRecord("r2", "task-b", 1, now.Add(-1*time.Minute))

	if err := st.Save(r1); err != nil {
		t.Fatal(err)
	}
	if err := st.Save(r2); err != nil {
		t.Fatal(err)
	}

	records, err := st.Recent(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	// Recent returns newest first.
	if records[0].ID != "r2" {
		t.Errorf("expected r2 first (newest), got %s", records[0].ID)
	}
}

func TestRecentLimit(t *testing.T) {
	st := openTestStore(t)
	now := time.Now()

	for i := 0; i < 5; i++ {
		r := makeRecord(fmt.Sprintf("r%d", i), "task", 0, now.Add(time.Duration(i)*time.Minute))
		if err := st.Save(r); err != nil {
			t.Fatal(err)
		}
	}

	records, err := st.Recent(3)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 3 {
		t.Errorf("expected 3 records with limit=3, got %d", len(records))
	}
}

func TestRecentNoLimit(t *testing.T) {
	st := openTestStore(t)
	now := time.Now()

	for i := 0; i < 5; i++ {
		r := makeRecord(fmt.Sprintf("r%d", i), "task", 0, now.Add(time.Duration(i)*time.Minute))
		st.Save(r)
	}

	records, err := st.Recent(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 5 {
		t.Errorf("expected 5 records with limit=0, got %d", len(records))
	}
}

func TestRecentEmptyStore(t *testing.T) {
	st := openTestStore(t)
	records, err := st.Recent(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records from empty store, got %d", len(records))
	}
}

func TestPruneByCount(t *testing.T) {
	st := openTestStore(t)
	now := time.Now()

	for i := 0; i < 5; i++ {
		r := makeRecord(fmt.Sprintf("r%d", i), "task", 0, now.Add(time.Duration(i)*time.Minute))
		st.Save(r)
	}

	if err := st.Prune(3); err != nil {
		t.Fatal(err)
	}

	records, err := st.Recent(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 3 {
		t.Errorf("expected 3 records after Prune(3), got %d", len(records))
	}
}

func TestPruneNoOpWhenUnderLimit(t *testing.T) {
	st := openTestStore(t)
	now := time.Now()

	for i := 0; i < 3; i++ {
		st.Save(makeRecord(fmt.Sprintf("r%d", i), "task", 0, now))
	}

	if err := st.Prune(10); err != nil {
		t.Fatal(err)
	}

	records, err := st.Recent(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 3 {
		t.Errorf("expected 3 records (no pruning), got %d", len(records))
	}
}

func TestPruneNoOpWhenZero(t *testing.T) {
	st := openTestStore(t)
	st.Save(makeRecord("r1", "task", 0, time.Now()))

	if err := st.Prune(0); err != nil {
		t.Fatal(err)
	}

	records, _ := st.Recent(0)
	if len(records) != 1 {
		t.Errorf("Prune(0) should be a no-op, got %d records", len(records))
	}
}

func TestPruneByAge(t *testing.T) {
	st := openTestStore(t)
	now := time.Now()

	old := makeRecord("old", "task", 0, now.AddDate(0, 0, -10))
	recent := makeRecord("recent", "task", 0, now)

	st.Save(old)
	st.Save(recent)

	if err := st.PruneByAge(5); err != nil {
		t.Fatal(err)
	}

	records, err := st.Recent(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record after PruneByAge(5), got %d", len(records))
	}
	if records[0].ID != "recent" {
		t.Errorf("expected 'recent' to survive, got %s", records[0].ID)
	}
}

func TestPruneByAgeNoOpWhenZero(t *testing.T) {
	st := openTestStore(t)
	st.Save(makeRecord("r1", "task", 0, time.Now().AddDate(0, 0, -100)))

	if err := st.PruneByAge(0); err != nil {
		t.Fatal(err)
	}

	records, _ := st.Recent(0)
	if len(records) != 1 {
		t.Errorf("PruneByAge(0) should be a no-op, got %d records", len(records))
	}
}

func TestClearAll(t *testing.T) {
	st := openTestStore(t)
	now := time.Now()

	for i := 0; i < 3; i++ {
		st.Save(makeRecord(fmt.Sprintf("r%d", i), "task", 0, now))
	}

	if err := st.ClearAll(); err != nil {
		t.Fatal(err)
	}

	records, err := st.Recent(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records after ClearAll, got %d", len(records))
	}
}

func TestSaveAndLoadDisabledSchedules(t *testing.T) {
	st := openTestStore(t)

	disabled := map[string]bool{
		"nightly": true,
		"hourly":  false,
	}

	if err := st.SaveDisabledSchedules(disabled); err != nil {
		t.Fatal(err)
	}

	loaded, err := st.LoadDisabledSchedules()
	if err != nil {
		t.Fatal(err)
	}

	if loaded["nightly"] != true {
		t.Error("expected nightly=true")
	}
	if loaded["hourly"] != false {
		t.Error("expected hourly=false")
	}
}

func TestLoadDisabledSchedulesEmptyStore(t *testing.T) {
	st := openTestStore(t)
	disabled, err := st.LoadDisabledSchedules()
	if err != nil {
		t.Fatal(err)
	}
	if len(disabled) != 0 {
		t.Errorf("expected empty map from fresh store, got %v", disabled)
	}
}

func TestSavePreservesAllFields(t *testing.T) {
	st := openTestStore(t)
	now := time.Now().Truncate(time.Millisecond) // bbolt round-trips via JSON

	r := &RunRecord{
		ID:         "test-id",
		TaskName:   "my-task",
		StartTime:  now,
		EndTime:    now.Add(5 * time.Second),
		DurationMs: 5000,
		ExitCode:   42,
		Trigger:    "scheduled",
		LogTail:    []string{"line1", "line2"},
	}
	if err := st.Save(r); err != nil {
		t.Fatal(err)
	}

	records, err := st.Recent(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	got := records[0]
	if got.ID != r.ID {
		t.Errorf("ID: got %q, want %q", got.ID, r.ID)
	}
	if got.TaskName != r.TaskName {
		t.Errorf("TaskName: got %q, want %q", got.TaskName, r.TaskName)
	}
	if got.ExitCode != r.ExitCode {
		t.Errorf("ExitCode: got %d, want %d", got.ExitCode, r.ExitCode)
	}
	if got.Trigger != r.Trigger {
		t.Errorf("Trigger: got %q, want %q", got.Trigger, r.Trigger)
	}
	if len(got.LogTail) != 2 {
		t.Errorf("LogTail: got %v", got.LogTail)
	}
}
