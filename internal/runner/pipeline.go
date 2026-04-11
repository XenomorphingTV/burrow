package runner

import (
	"fmt"

	"github.com/xenomorphingtv/burrow/internal/config"
)

// Resolve returns the ordered task execution list for the target task,
// using DFS topological sort. The target task is included at the end.
// Returns an error if a cycle is detected or a dependency is missing.
func Resolve(target string, tasks map[string]config.Task) ([]string, error) {
	// States: 0=unvisited, 1=visiting, 2=visited
	state := make(map[string]int)
	var order []string

	var dfs func(name string) error
	dfs = func(name string) error {
		switch state[name] {
		case 1:
			return fmt.Errorf("cycle detected involving task %q", name)
		case 2:
			return nil
		}

		if _, ok := tasks[name]; !ok {
			return fmt.Errorf("task %q not found", name)
		}

		state[name] = 1
		task := tasks[name]
		for _, dep := range task.DependsOn {
			if err := dfs(dep); err != nil {
				return err
			}
		}
		state[name] = 2
		order = append(order, name)
		return nil
	}

	if err := dfs(target); err != nil {
		return nil, err
	}
	return order, nil
}
