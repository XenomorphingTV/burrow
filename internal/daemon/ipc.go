package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sort"
	"time"

	"github.com/xenomorphingtv/burrow/internal/store"
)

// Request is a command sent to the daemon over the Unix socket.
type Request struct {
	Cmd     string          `json:"cmd"`
	Task    string          `json:"task,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Response is the daemon's reply.
type Response struct {
	OK    bool            `json:"ok"`
	Error string          `json:"error,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
}

// StatusData is the payload of a "status" response.
type StatusData struct {
	PID           int            `json:"pid"`
	StartTime     time.Time      `json:"start_time"`
	Uptime        string         `json:"uptime"`
	LastHeartbeat time.Time      `json:"last_heartbeat"`
	ConfigMtime   time.Time      `json:"config_mtime"`
	NextRuns      []NextRunEntry `json:"next_runs"`
}

// NextRunEntry describes the next scheduled run for one schedule.
type NextRunEntry struct {
	Schedule string    `json:"schedule"`
	Task     string    `json:"task"`
	Spec     string    `json:"spec"`
	Next     time.Time `json:"next"`
}

// handleConn services a single IPC connection.
func handleConn(conn net.Conn, d *daemon) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second)) //nolint:errcheck

	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return
	}

	var req Request
	if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
		writeResponse(conn, Response{OK: false, Error: "bad request: " + err.Error()})
		return
	}

	switch req.Cmd {
	case "status":
		writeResponse(conn, buildStatusResponse(d))
	case "reload":
		err := d.reload()
		if err != nil {
			writeResponse(conn, Response{OK: false, Error: err.Error()})
		} else {
			writeResponse(conn, Response{OK: true})
		}
	case "run":
		if req.Task == "" {
			writeResponse(conn, Response{OK: false, Error: "task name required"})
			return
		}
		d.mu.RLock()
		_, ok := d.cfg.Tasks[req.Task]
		d.mu.RUnlock()
		if !ok {
			writeResponse(conn, Response{OK: false, Error: fmt.Sprintf("task %q not found", req.Task)})
			return
		}
		go d.onScheduledRun(req.Task, "manual")
		writeResponse(conn, Response{OK: true})
	case "history_save":
		var rec store.RunRecord
		if err := json.Unmarshal(req.Payload, &rec); err != nil {
			writeResponse(conn, Response{OK: false, Error: "bad payload: " + err.Error()})
			return
		}
		if err := d.st.Save(&rec); err != nil {
			writeResponse(conn, Response{OK: false, Error: err.Error()})
		} else {
			writeResponse(conn, Response{OK: true})
		}

	case "history_recent":
		var limit int
		json.Unmarshal(req.Payload, &limit) //nolint:errcheck — zero is a valid fallback
		records, err := d.st.Recent(limit)
		if err != nil {
			writeResponse(conn, Response{OK: false, Error: err.Error()})
			return
		}
		raw, _ := json.Marshal(records)
		writeResponse(conn, Response{OK: true, Data: raw})

	case "history_clear":
		if err := d.st.ClearAll(); err != nil {
			writeResponse(conn, Response{OK: false, Error: err.Error()})
		} else {
			writeResponse(conn, Response{OK: true})
		}

	case "schedules_save":
		var disabled map[string]bool
		if err := json.Unmarshal(req.Payload, &disabled); err != nil {
			writeResponse(conn, Response{OK: false, Error: "bad payload: " + err.Error()})
			return
		}
		if err := d.st.SaveDisabledSchedules(disabled); err != nil {
			writeResponse(conn, Response{OK: false, Error: err.Error()})
		} else {
			d.mu.Lock()
			applyDisabledSchedules(d.sched, disabled)
			d.mu.Unlock()
			writeResponse(conn, Response{OK: true})
		}

	case "schedules_load":
		disabled, err := d.st.LoadDisabledSchedules()
		if err != nil {
			writeResponse(conn, Response{OK: false, Error: err.Error()})
			return
		}
		raw, _ := json.Marshal(disabled)
		writeResponse(conn, Response{OK: true, Data: raw})

	default:
		writeResponse(conn, Response{OK: false, Error: "unknown command: " + req.Cmd})
	}
}

func buildStatusResponse(d *daemon) Response {
	d.mu.RLock()
	startTime := d.startTime
	heartbeat := d.heartbeat
	cfgMtime := d.cfgMtime
	schedEntries := d.sched.AllSchedules()
	d.mu.RUnlock()

	uptime := time.Since(startTime).Truncate(time.Second).String()

	var nextRuns []NextRunEntry
	for _, e := range schedEntries {
		if e.Next != nil {
			nextRuns = append(nextRuns, NextRunEntry{
				Schedule: e.Name,
				Task:     e.TaskName,
				Spec:     e.Spec,
				Next:     *e.Next,
			})
		}
	}
	sort.Slice(nextRuns, func(i, j int) bool {
		return nextRuns[i].Next.Before(nextRuns[j].Next)
	})

	data := StatusData{
		PID:           os.Getpid(),
		StartTime:     startTime,
		Uptime:        uptime,
		LastHeartbeat: heartbeat,
		ConfigMtime:   cfgMtime,
		NextRuns:      nextRuns,
	}
	raw, _ := json.Marshal(data)
	return Response{OK: true, Data: raw}
}

func writeResponse(conn net.Conn, r Response) {
	data, _ := json.Marshal(r)
	data = append(data, '\n')
	conn.Write(data) //nolint:errcheck
}

// Client communicates with the running daemon.
type Client struct {
	conn net.Conn
}

// Connect opens a connection to the daemon socket.
// Returns an error if the daemon is not running.
func Connect() (*Client, error) {
	conn, err := net.DialTimeout("unix", SocketPath(), 3*time.Second)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to daemon (is it running?): %w", err)
	}
	return &Client{conn: conn}, nil
}

// Close closes the connection.
func (c *Client) Close() {
	c.conn.Close()
}

// Status requests the daemon's status.
func (c *Client) Status() (*StatusData, error) {
	resp, err := c.send(Request{Cmd: "status"})
	if err != nil {
		return nil, err
	}
	var data StatusData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("parse status: %w", err)
	}
	return &data, nil
}

// Reload tells the daemon to reload its config.
func (c *Client) Reload() error {
	_, err := c.send(Request{Cmd: "reload"})
	return err
}

// Run asks the daemon to run a task immediately.
func (c *Client) Run(taskName string) error {
	_, err := c.send(Request{Cmd: "run", Task: taskName})
	return err
}

func (c *Client) send(req Request) (*Response, error) {
	c.conn.SetDeadline(time.Now().Add(10 * time.Second)) //nolint:errcheck

	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	if _, err := c.conn.Write(data); err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	scanner := bufio.NewScanner(c.conn)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}
		return nil, fmt.Errorf("daemon closed connection without reply")
	}

	var resp Response
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if !resp.OK {
		return nil, fmt.Errorf("daemon error: %s", resp.Error)
	}
	return &resp, nil
}
