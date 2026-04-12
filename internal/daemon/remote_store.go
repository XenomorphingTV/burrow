package daemon

import (
	"encoding/json"
	"fmt"

	"github.com/XenomorphingTV/burrow/internal/store"
)

// RemoteStore implements store.Storer by forwarding every call to the daemon
// over the Unix socket. It is used by the TUI and executor when the daemon is
// already running and holds the bbolt lock.
//
// Each method opens a fresh connection, sends one request, and closes it.
// This is fine for a personal tool where these calls are infrequent.
type RemoteStore struct{}

// NewRemoteStore returns a RemoteStore that talks to the daemon socket.
func NewRemoteStore() *RemoteStore { return &RemoteStore{} }

func (r *RemoteStore) dial() (*Client, error) {
	c, err := Connect()
	if err != nil {
		return nil, fmt.Errorf("remote store: %w", err)
	}
	return c, nil
}

// Save forwards a run record to the daemon for persistence.
func (r *RemoteStore) Save(rec *store.RunRecord) error {
	c, err := r.dial()
	if err != nil {
		return err
	}
	defer c.Close()

	payload, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	_, err = c.send(Request{Cmd: "history_save", Payload: payload})
	return err
}

// Recent asks the daemon for the most recent run records.
func (r *RemoteStore) Recent(limit int) ([]*store.RunRecord, error) {
	c, err := r.dial()
	if err != nil {
		return nil, err
	}
	defer c.Close()

	payload, _ := json.Marshal(limit)
	resp, err := c.send(Request{Cmd: "history_recent", Payload: payload})
	if err != nil {
		return nil, err
	}
	var records []*store.RunRecord
	if err := json.Unmarshal(resp.Data, &records); err != nil {
		return nil, fmt.Errorf("parse records: %w", err)
	}
	return records, nil
}

// ClearAll asks the daemon to delete all run records.
func (r *RemoteStore) ClearAll() error {
	c, err := r.dial()
	if err != nil {
		return err
	}
	defer c.Close()
	_, err = c.send(Request{Cmd: "history_clear"})
	return err
}

// SaveDisabledSchedules forwards the disabled schedule map to the daemon.
func (r *RemoteStore) SaveDisabledSchedules(disabled map[string]bool) error {
	c, err := r.dial()
	if err != nil {
		return err
	}
	defer c.Close()

	payload, err := json.Marshal(disabled)
	if err != nil {
		return err
	}
	_, err = c.send(Request{Cmd: "schedules_save", Payload: payload})
	return err
}

// LoadDisabledSchedules reads the disabled schedule map from the daemon.
// Returns an empty map (not an error) if the daemon is unreachable, so the
// TUI starts cleanly even when the socket is briefly unavailable.
func (r *RemoteStore) LoadDisabledSchedules() (map[string]bool, error) {
	c, err := r.dial()
	if err != nil {
		return make(map[string]bool), nil
	}
	defer c.Close()

	resp, err := c.send(Request{Cmd: "schedules_load"})
	if err != nil {
		return make(map[string]bool), nil
	}
	var disabled map[string]bool
	if err := json.Unmarshal(resp.Data, &disabled); err != nil {
		return make(map[string]bool), nil
	}
	return disabled, nil
}
