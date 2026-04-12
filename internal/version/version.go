// Package version holds the build-time version string.
// The Makefile injects the real value via -ldflags; the default is "dev".
package version

var Version = "0.1.3"
