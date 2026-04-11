package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
)

const bucketRuns = "runs"
const bucketSettings = "settings"
const keyDisabledSchedules = "disabled_schedules"

// Storer is the interface used by the executor and TUI.
// It is satisfied by both the local bbolt Store and the daemon RemoteStore.
type Storer interface {
	Save(r *RunRecord) error
	Recent(limit int) ([]*RunRecord, error)
	ClearAll() error
	SaveDisabledSchedules(disabled map[string]bool) error
	LoadDisabledSchedules() (map[string]bool, error)
}

// RunRecord stores the result of a single task run.
type RunRecord struct {
	ID         string    `json:"id"`
	TaskName   string    `json:"task_name"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	DurationMs int64     `json:"duration_ms"`
	ExitCode   int       `json:"exit_code"`
	Trigger    string    `json:"trigger"`  // "manual" | "scheduled" | "pipeline"
	LogTail    []string  `json:"log_tail"` // last 200 lines
}

// Store wraps a bbolt database for run history.
type Store struct {
	db *bolt.DB
}

// Open opens the history database at the given directory.
func Open(dir string) (*Store, error) {
	path := filepath.Join(dir, "history.db")
	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("open history db: %w", err)
	}

	// Create buckets if they don't exist
	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(bucketRuns)); err != nil {
			return err
		}
		_, err := tx.CreateBucketIfNotExists([]byte(bucketSettings))
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create bucket: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the underlying database.
func (s *Store) Close() error {
	return s.db.Close()
}

// SaveDisabledSchedules persists the set of disabled schedule names.
func (s *Store) SaveDisabledSchedules(disabled map[string]bool) error {
	data, err := json.Marshal(disabled)
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketSettings))
		return b.Put([]byte(keyDisabledSchedules), data)
	})
}

// LoadDisabledSchedules returns the previously persisted set of disabled schedule names.
func (s *Store) LoadDisabledSchedules() (map[string]bool, error) {
	disabled := make(map[string]bool)
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketSettings))
		v := b.Get([]byte(keyDisabledSchedules))
		if v == nil {
			return nil
		}
		return json.Unmarshal(v, &disabled)
	})
	return disabled, err
}

// Save persists a RunRecord to the database.
func (s *Store) Save(r *RunRecord) error {
	data, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("marshal record: %w", err)
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketRuns))
		return b.Put([]byte(r.ID), data)
	})
}

// Recent returns the most recent limit run records, sorted newest first.
func (s *Store) Recent(limit int) ([]*RunRecord, error) {
	var records []*RunRecord
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketRuns))
		return b.ForEach(func(k, v []byte) error {
			var r RunRecord
			if err := json.Unmarshal(v, &r); err != nil {
				return nil // skip bad records
			}
			records = append(records, &r)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}

	// Sort newest first
	sort.Slice(records, func(i, j int) bool {
		return records[i].StartTime.After(records[j].StartTime)
	})

	if limit > 0 && len(records) > limit {
		records = records[:limit]
	}
	return records, nil
}

// Prune removes the oldest records beyond maxRun, keeping the most recent.
// No-op if maxRun <= 0.
func (s *Store) Prune(maxRun int) error {
	if maxRun <= 0 {
		return nil
	}
	records, err := s.Recent(0) // all records, newest first
	if err != nil || len(records) <= maxRun {
		return err
	}
	toDelete := records[maxRun:]
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketRuns))
		for _, r := range toDelete {
			if err := b.Delete([]byte(r.ID)); err != nil {
				return err
			}
		}
		return nil
	})
}

// PruneByAge removes history records older than maxAgeDays days.
// No-op if maxAgeDays <= 0.
func (s *Store) PruneByAge(maxAgeDays int) error {
	if maxAgeDays <= 0 {
		return nil
	}
	cutoff := time.Now().AddDate(0, 0, -maxAgeDays)
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketRuns))
		var toDelete [][]byte
		b.ForEach(func(k, v []byte) error {
			var r RunRecord
			if err := json.Unmarshal(v, &r); err != nil {
				return nil
			}
			if r.StartTime.Before(cutoff) {
				toDelete = append(toDelete, append([]byte{}, k...))
			}
			return nil
		})
		for _, k := range toDelete {
			b.Delete(k)
		}
		return nil
	})
}

// PruneLogFiles removes log files in logDir older than maxAgeDays days.
// No-op if maxAgeDays <= 0.
func PruneLogFiles(logDir string, maxAgeDays int) error {
	if maxAgeDays <= 0 {
		return nil
	}
	logDir = expandPath(logDir)
	entries, err := os.ReadDir(logDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	cutoff := time.Now().AddDate(0, 0, -maxAgeDays)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(logDir, entry.Name()))
		}
	}
	return nil
}

// expandPath expands a leading ~ to the user home directory.
func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

// ClearAll deletes all run records from the database.
func (s *Store) ClearAll() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		if err := tx.DeleteBucket([]byte(bucketRuns)); err != nil {
			return err
		}
		_, err := tx.CreateBucket([]byte(bucketRuns))
		return err
	})
}

// LastRun returns the most recent run record for a task name.
func (s *Store) LastRun(taskName string) (*RunRecord, error) {
	var latest *RunRecord
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketRuns))
		return b.ForEach(func(k, v []byte) error {
			var r RunRecord
			if err := json.Unmarshal(v, &r); err != nil {
				return nil
			}
			if r.TaskName == taskName {
				if latest == nil || r.StartTime.After(latest.StartTime) {
					latest = &r
				}
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return latest, nil
}
