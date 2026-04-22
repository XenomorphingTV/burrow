package main

import (
	"fmt"
	"os"
)

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		if err := runTUI(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	switch args[0] {
	case "help", "--help", "-h":
		printHelp()
	case "version", "--version", "-v":
		printVersion()
	case "init":
		if err := runInit(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "run":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: burrow run <task-name>")
			os.Exit(1)
		}
		if err := runHeadless(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "list":
		var jsonOut bool
		var outFile string
		for _, a := range args[1:] {
			if a == "--json" {
				jsonOut = true
			} else {
				outFile = a
			}
		}
		if err := runList(jsonOut, outFile); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "check":
		runCheck()
	case "daemon":
		if err := runDaemon(args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "Run 'burrow help' for usage.")
		os.Exit(1)
	}
}
