package v2

import (
	"context"
	"time"

	"github.com/XenomorphingTV/burrow/internal/runner"
	"github.com/XenomorphingTV/burrow/internal/store"
	"github.com/XenomorphingTV/burrow/internal/tui/v2/messages"
	tea "github.com/charmbracelet/bubbletea"
)

// historyLoadedMsg carries loaded history records.
type historyLoadedMsg []*store.RunRecord

// statsLoadedMsg carries all run records for the stats tab.
type statsLoadedMsg []*store.RunRecord

// loadHistory is a command that loads recent history from the store.
func loadHistory(st store.Storer) tea.Cmd {
	return func() tea.Msg {
		if st == nil {
			return historyLoadedMsg(nil)
		}
		records, err := st.Recent(100)
		if err != nil {
			return historyLoadedMsg(nil)
		}
		return historyLoadedMsg(records)
	}
}

// loadAllHistory loads every run record for stats computation.
func loadAllHistory(st store.Storer) tea.Cmd {
	return func() tea.Msg {
		if st == nil {
			return statsLoadedMsg(nil)
		}
		records, err := st.Recent(0) // 0 = no limit
		if err != nil {
			return statsLoadedMsg(nil)
		}
		return statsLoadedMsg(records)
	}
}

// awaitLog blocks on a log channel and returns the next line as a tea.Msg.
func awaitLog(ch <-chan runner.LogLine) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return runner.LogLine{Done: true, ExitCode: -1}
		}
		return runner.LogLine(line)
	}
}

// Returns a tea.Cmd that blocks until any file matching patterns
// (rooted at baseDir) changes, then emits watchTriggeredMsg. It exits silently
// when ctx is cancelled.
func startWatcher(ctx context.Context, taskName string, patterns []string, baseDir string) tea.Cmd {
	return func() tea.Msg {
		snap := runner.SnapshotFiles(patterns, baseDir)
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(500 * time.Millisecond):
				newSnap := runner.SnapshotFiles(patterns, baseDir)
				if !runner.SnapshotsMatch(snap, newSnap) {
					return messages.WatchTriggeredMessage{TaskName: taskName}
				}
			}
		}
	}
}
