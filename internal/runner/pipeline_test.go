package runner

import (
	"strings"
	"testing"

	"github.com/XenomorphingTV/burrow/internal/config"
)

func TestResolveSingleTask(t *testing.T) {
	tasks := map[string]config.Task{
		"hello": {Cmd: "echo hello"},
	}
	order, err := Resolve("hello", tasks)
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 1 || order[0] != "hello" {
		t.Errorf("expected [hello], got %v", order)
	}
}

func TestResolveChain(t *testing.T) {
	tasks := map[string]config.Task{
		"build": {Cmd: "make build", DependsOn: []string{"lint"}},
		"lint":  {Cmd: "golint ./..."},
	}
	order, err := Resolve("build", tasks)
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 {
		t.Fatalf("expected 2 tasks, got %d: %v", len(order), order)
	}
	if order[0] != "lint" || order[1] != "build" {
		t.Errorf("expected [lint, build], got %v", order)
	}
}

func TestResolveDAGDiamondDependency(t *testing.T) {
	// deploy -> test, lint; test -> nothing; lint -> nothing
	tasks := map[string]config.Task{
		"deploy": {Cmd: "deploy.sh", DependsOn: []string{"test", "lint"}},
		"test":   {Cmd: "go test ./..."},
		"lint":   {Cmd: "golint ./..."},
	}
	order, err := Resolve("deploy", tasks)
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 3 {
		t.Fatalf("expected 3 tasks, got %d: %v", len(order), order)
	}
	if order[len(order)-1] != "deploy" {
		t.Errorf("deploy should be last, got %v", order)
	}
}

func TestResolveDeepChain(t *testing.T) {
	// d -> c -> b -> a
	tasks := map[string]config.Task{
		"a": {Cmd: "a"},
		"b": {Cmd: "b", DependsOn: []string{"a"}},
		"c": {Cmd: "c", DependsOn: []string{"b"}},
		"d": {Cmd: "d", DependsOn: []string{"c"}},
	}
	order, err := Resolve("d", tasks)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a", "b", "c", "d"}
	if len(order) != len(want) {
		t.Fatalf("expected %v, got %v", want, order)
	}
	for i, name := range want {
		if order[i] != name {
			t.Errorf("position %d: expected %q, got %q", i, name, order[i])
		}
	}
}

func TestResolveCycleDetected(t *testing.T) {
	tasks := map[string]config.Task{
		"a": {Cmd: "a", DependsOn: []string{"b"}},
		"b": {Cmd: "b", DependsOn: []string{"a"}},
	}
	_, err := Resolve("a", tasks)
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("expected 'cycle' in error message, got: %v", err)
	}
}

func TestResolveMissingDependency(t *testing.T) {
	tasks := map[string]config.Task{
		"deploy": {Cmd: "deploy.sh", DependsOn: []string{"nonexistent"}},
	}
	_, err := Resolve("deploy", tasks)
	if err == nil {
		t.Fatal("expected error for missing dependency, got nil")
	}
}

func TestResolveUnknownTarget(t *testing.T) {
	_, err := Resolve("nonexistent", map[string]config.Task{})
	if err == nil {
		t.Fatal("expected error for unknown target, got nil")
	}
}

func TestResolveNoDuplicates(t *testing.T) {
	// Both b and c depend on a; a should appear only once.
	tasks := map[string]config.Task{
		"a": {Cmd: "a"},
		"b": {Cmd: "b", DependsOn: []string{"a"}},
		"c": {Cmd: "c", DependsOn: []string{"a", "b"}},
	}
	order, err := Resolve("c", tasks)
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	for _, name := range order {
		if seen[name] {
			t.Errorf("task %q appears more than once in %v", name, order)
		}
		seen[name] = true
	}
}
