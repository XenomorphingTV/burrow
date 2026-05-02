package messages

type RunTaskMessage struct {
	Name    string
	Trigger string
	Env     map[string]string
}

type KillTaskMessage struct {
	Name string
}

type ToggleScheduleMessage struct {
	Name string
	Kind string
}

type EditCronMessage struct {
	Name string
	Cron string
}

type ClearHistoryMessage struct {
}

// WatchTriggeredMsg is sent when a watched file changes.
type WatchTriggeredMessage struct {
	TaskName string
}
