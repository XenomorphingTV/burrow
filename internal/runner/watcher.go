package runner

import (
	"os"
	"path/filepath"
	"strings"
)

// FileInfo records the mtime and size of a single file.
type FileInfo struct {
	Mtime int64
	Size  int64
}

// SnapshotFiles returns a map of absolute path → FileInfo for every file
// matching any of the given glob patterns. Patterns are resolved relative to
// baseDir. "**" in a pattern causes a recursive directory walk; the part after
// "**/" is matched against each file's base name via filepath.Match.
func SnapshotFiles(patterns []string, baseDir string) map[string]FileInfo {
	baseDir = expandHome(baseDir)
	snap := make(map[string]FileInfo)
	for _, pattern := range patterns {
		if !filepath.IsAbs(pattern) && baseDir != "" {
			pattern = filepath.Join(baseDir, pattern)
		}
		for _, p := range expandGlob(pattern) {
			info, err := os.Stat(p)
			if err != nil || info.IsDir() {
				continue
			}
			snap[p] = FileInfo{Mtime: info.ModTime().UnixNano(), Size: info.Size()}
		}
	}
	return snap
}

// SnapshotsMatch returns true when a and b contain identical entries.
func SnapshotsMatch(a, b map[string]FileInfo) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		if vb, ok := b[k]; !ok || va != vb {
			return false
		}
	}
	return true
}

// expandGlob expands a single glob pattern, supporting "**" for recursive
// matching. Without "**" it delegates to filepath.Glob.
func expandGlob(pattern string) []string {
	if !strings.Contains(pattern, "**") {
		matches, _ := filepath.Glob(pattern)
		return matches
	}

	idx := strings.Index(pattern, "**")
	root := pattern[:idx]
	if root == "" {
		root = "."
	} else {
		root = filepath.Clean(root)
	}

	// The part after "**" (strip leading separator).
	suffix := strings.TrimPrefix(pattern[idx+2:], string(filepath.Separator))

	var results []string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if suffix == "" {
			results = append(results, path)
			return nil
		}
		if matched, _ := filepath.Match(suffix, filepath.Base(path)); matched {
			results = append(results, path)
		}
		return nil
	})
	return results
}
