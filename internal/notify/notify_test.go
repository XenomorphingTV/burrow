package notify

import "testing"

func TestShouldNotifyTaskListTakesPrecedence(t *testing.T) {
	// When the task has its own notify list, global is ignored.
	if !ShouldNotify(EventSuccess, []string{"success"}, []string{"failure"}) {
		t.Error("task notify=success should fire on success")
	}
	if ShouldNotify(EventFailure, []string{"success"}, []string{"failure"}) {
		t.Error("task notify=success should not fire on failure (global ignored)")
	}
}

func TestShouldNotifyFallsBackToGlobal(t *testing.T) {
	if !ShouldNotify(EventFailure, nil, []string{"failure"}) {
		t.Error("global notify=failure should fire on failure when task list is nil")
	}
	if ShouldNotify(EventSuccess, nil, []string{"failure"}) {
		t.Error("global notify=failure should not fire on success")
	}
}

func TestShouldNotifyEmptyTaskListFallsBackToGlobal(t *testing.T) {
	if !ShouldNotify(EventSuccess, []string{}, []string{"success"}) {
		t.Error("empty task list should fall back to global")
	}
}

func TestShouldNotifyAlwaysMatchesAnyEvent(t *testing.T) {
	if !ShouldNotify(EventSuccess, []string{"always"}, nil) {
		t.Error("'always' should fire on success")
	}
	if !ShouldNotify(EventFailure, []string{"always"}, nil) {
		t.Error("'always' should fire on failure")
	}
}

func TestShouldNotifyAlwaysInGlobal(t *testing.T) {
	if !ShouldNotify(EventSuccess, nil, []string{"always"}) {
		t.Error("global 'always' should fire on success")
	}
	if !ShouldNotify(EventFailure, nil, []string{"always"}) {
		t.Error("global 'always' should fire on failure")
	}
}

func TestShouldNotifyEmptyListsNeverFire(t *testing.T) {
	if ShouldNotify(EventSuccess, nil, nil) {
		t.Error("nil lists should not fire")
	}
	if ShouldNotify(EventFailure, []string{}, []string{}) {
		t.Error("empty lists should not fire")
	}
}

func TestShouldNotifyBothEvents(t *testing.T) {
	both := []string{EventSuccess, EventFailure}
	if !ShouldNotify(EventSuccess, both, nil) {
		t.Error("success+failure list should fire on success")
	}
	if !ShouldNotify(EventFailure, both, nil) {
		t.Error("success+failure list should fire on failure")
	}
}

func TestShouldNotifyUnknownEventDoesNotMatch(t *testing.T) {
	if ShouldNotify("unknown-event", []string{"success", "failure"}, nil) {
		t.Error("unknown event should not match a specific list")
	}
}
