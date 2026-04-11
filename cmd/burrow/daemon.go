package main

import (
	"fmt"
	"os"
	"time"

	"github.com/xenomorphingtv/burrow/internal/daemon"
)

func runDaemon(args []string) error {
	sub := ""
	if len(args) > 0 {
		sub = args[0]
	}

	switch sub {
	case "start":
		return daemonStart()
	case "stop":
		return daemonStop()
	case "status":
		return daemonStatus()
	case "install":
		return daemonInstall()
	case "uninstall":
		return daemonUninstall()
	case "_serve":
		// Hidden subcommand invoked by the OS service file.
		return daemon.Serve()
	default:
		fmt.Fprintln(os.Stderr, "usage: burrow daemon <start|stop|status|install|uninstall>")
		os.Exit(1)
		return nil
	}
}

func daemonInstall() error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	if err := daemon.Install(execPath); err != nil {
		return err
	}
	fmt.Printf("Service file written (%s).\n", daemon.ServiceName())
	fmt.Printf("Run 'burrow daemon start' to enable and start it.\n")
	return nil
}

func daemonStart() error {
	if pid, running := daemon.IsRunning(); running {
		fmt.Printf("Daemon is already running (pid %d).\n", pid)
		return nil
	}
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	if err := daemon.StartService(execPath); err != nil {
		return fmt.Errorf("start service: %w", err)
	}
	fmt.Println("Daemon started.")
	return nil
}

func daemonStop() error {
	if _, running := daemon.IsRunning(); !running {
		fmt.Println("Daemon is not running.")
		return nil
	}
	if err := daemon.StopService(); err != nil {
		return fmt.Errorf("stop service: %w", err)
	}
	fmt.Println("Daemon stopped.")
	return nil
}

func daemonStatus() error {
	pid, running := daemon.IsRunning()
	if !running {
		fmt.Println("Status:  stopped")
		fmt.Printf("PID file: %s\n", daemon.PIDPath())
		fmt.Printf("Socket:   %s\n", daemon.SocketPath())
		return nil
	}

	fmt.Printf("Status:  running (pid %d)\n", pid)

	client, err := daemon.Connect()
	if err != nil {
		fmt.Printf("Socket:  %s (unreachable: %v)\n", daemon.SocketPath(), err)
		return nil
	}
	defer client.Close()

	status, err := client.Status()
	if err != nil {
		fmt.Printf("IPC error: %v\n", err)
		return nil
	}

	fmt.Printf("Uptime:  %s\n", status.Uptime)
	fmt.Printf("Started: %s\n", status.StartTime.Local().Format(time.RFC1123))
	fmt.Printf("Heartbeat: %s ago\n", time.Since(status.LastHeartbeat).Truncate(time.Second))

	if !status.ConfigMtime.IsZero() {
		fmt.Printf("Config:  loaded %s\n", status.ConfigMtime.Local().Format(time.RFC1123))
	}

	if len(status.NextRuns) == 0 {
		fmt.Println("Schedules: none")
	} else {
		fmt.Println("Next scheduled runs:")
		for _, r := range status.NextRuns {
			in := time.Until(r.Next).Truncate(time.Second)
			fmt.Printf("  %-20s  %-16s  %s  (in %s)\n",
				r.Schedule, r.Task, r.Next.Local().Format("2006-01-02 15:04:05"), in)
		}
	}
	return nil
}

func daemonUninstall() error {
	if err := daemon.Uninstall(); err != nil {
		return err
	}
	fmt.Println("Service uninstalled.")
	return nil
}
