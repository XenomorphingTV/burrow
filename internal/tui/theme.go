package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Theme holds the color palette used to render the TUI.
type Theme struct {
	Bg       string
	BgHeader string
	Border   string
	Selected string
	Text     string
	Dim      string
	Purple   string
	Blue     string
	Green    string
	Red      string
	Yellow   string
}

// themeFile is the TOML-decodable representation of a custom theme file.
type themeFile struct {
	Bg       string `toml:"bg"`
	BgHeader string `toml:"bg_header"`
	Border   string `toml:"border"`
	Selected string `toml:"selected"`
	Text     string `toml:"text"`
	Dim      string `toml:"dim"`
	Purple   string `toml:"purple"`
	Blue     string `toml:"blue"`
	Green    string `toml:"green"`
	Red      string `toml:"red"`
	Yellow   string `toml:"yellow"`
}

// Built-in named themes.
var (
	ThemeCatppuccinMocha = Theme{
		Bg:       "#1a1b1e",
		BgHeader: "#181820",
		Border:   "#2a2b35",
		Selected: "#2a2b3d",
		Text:     "#cdd6f4",
		Dim:      "#45475a",
		Purple:   "#cba6f7",
		Blue:     "#89b4fa",
		Green:    "#a6e3a1",
		Red:      "#f38ba8",
		Yellow:   "#f9e2af",
	}

	ThemeNord = Theme{
		Bg:       "#2e3440",
		BgHeader: "#242933",
		Border:   "#3b4252",
		Selected: "#434c5e",
		Text:     "#eceff4",
		Dim:      "#4c566a",
		Purple:   "#b48ead",
		Blue:     "#88c0d0",
		Green:    "#a3be8c",
		Red:      "#bf616a",
		Yellow:   "#ebcb8b",
	}

	ThemeDracula = Theme{
		Bg:       "#282a36",
		BgHeader: "#21222c",
		Border:   "#44475a",
		Selected: "#44475a",
		Text:     "#f8f8f2",
		Dim:      "#6272a4",
		Purple:   "#bd93f9",
		Blue:     "#8be9fd",
		Green:    "#50fa7b",
		Red:      "#ff5555",
		Yellow:   "#f1fa8c",
	}

	ThemeGruvbox = Theme{
		Bg:       "#282828",
		BgHeader: "#1d2021",
		Border:   "#3c3836",
		Selected: "#504945",
		Text:     "#ebdbb2",
		Dim:      "#665c54",
		Purple:   "#d3869b",
		Blue:     "#83a598",
		Green:    "#b8bb26",
		Red:      "#fb4934",
		Yellow:   "#fabd2f",
	}
)

var builtinThemes = map[string]Theme{
	"catppuccin-mocha": ThemeCatppuccinMocha,
	"nord":             ThemeNord,
	"dracula":          ThemeDracula,
	"gruvbox":          ThemeGruvbox,
}

// ResolveTheme returns the Theme for the given name or path.
// If name ends in ".toml" it is treated as a file path and loaded with
// LoadThemeFile. Unknown names and empty strings fall back to Catppuccin Mocha.
func ResolveTheme(name string) Theme {
	if strings.HasSuffix(name, ".toml") {
		t, err := LoadThemeFile(name)
		if err != nil {
			return ThemeCatppuccinMocha
		}
		return t
	}
	if t, ok := builtinThemes[name]; ok {
		return t
	}
	return ThemeCatppuccinMocha
}

// LoadThemeFile reads a TOML theme file and returns the resulting Theme.
// Any field left unset in the file retains the Catppuccin Mocha default,
// so partial theme files are valid.
func LoadThemeFile(path string) (Theme, error) {
	path = expandHome(path)

	var tf themeFile
	if _, err := toml.DecodeFile(path, &tf); err != nil {
		return Theme{}, fmt.Errorf("load theme file %q: %w", path, err)
	}

	// Start from the default and apply only the values the file specifies.
	base := ThemeCatppuccinMocha
	if tf.Bg != "" {
		base.Bg = tf.Bg
	}
	if tf.BgHeader != "" {
		base.BgHeader = tf.BgHeader
	}
	if tf.Border != "" {
		base.Border = tf.Border
	}
	if tf.Selected != "" {
		base.Selected = tf.Selected
	}
	if tf.Text != "" {
		base.Text = tf.Text
	}
	if tf.Dim != "" {
		base.Dim = tf.Dim
	}
	if tf.Purple != "" {
		base.Purple = tf.Purple
	}
	if tf.Blue != "" {
		base.Blue = tf.Blue
	}
	if tf.Green != "" {
		base.Green = tf.Green
	}
	if tf.Red != "" {
		base.Red = tf.Red
	}
	if tf.Yellow != "" {
		base.Yellow = tf.Yellow
	}
	return base, nil
}

func expandHome(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[2:])
}
