package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseScenes(t *testing.T) {
	content := `
# comment
evening: 10.0.0.1 on, @all color red
broken-line
morning:10.0.0.2 color blue , 10.0.0.3 brightness 20
`

	result := parseScenes(content)
	require.Len(t, result, 2)

	ev := result["evening"]
	require.Equal(t, []string{"10.0.0.1 on", "@all color red"}, ev.Commands)

	morning := result["morning"]
	require.Equal(t, []string{"10.0.0.2 color blue", "10.0.0.3 brightness 20"}, morning.Commands)
}

func TestResolveAliases(t *testing.T) {
	groups := map[string][]string{
		"@room":    {"10.0.0.1", "@kitchen"},
		"@kitchen": {"10.0.0.2", "10.0.0.3"},
		"@loop":    {"@loop"},
	}

	resolved := resolveAliases([]string{"@room", "10.0.0.1"}, groups)
	require.Equal(t, []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"}, resolved)
}

func TestReadFileFromDirs(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir2, "sample.txt"), []byte(" value \n"), 0o644))

	content, err := readFileFromDirs("sample.txt", dir1, dir2)
	require.NoError(t, err)
	require.Equal(t, "value", content)

	_, err = readFileFromDirs("missing.txt", dir1)
	require.Error(t, err)
}

func TestLoadBulbGroups(t *testing.T) {
	origExecDir := execDir
	execDir = t.TempDir()
	t.Cleanup(func() { execDir = origExecDir })

	fileContent := strings.TrimSpace(`
@room: 10.0.0.1 10.0.0.2
@kitchen: 10.0.0.3
@all: @room @kitchen 10.0.0.4
`)

	require.NoError(t, os.WriteFile(filepath.Join(execDir, "bulbs.txt"), []byte(fileContent), 0o644))

	groups := loadBulbGroups()
	require.Equal(t, "10.0.0.1 10.0.0.2", groups["@room"])
	require.Equal(t, "10.0.0.3", groups["@kitchen"])
	require.Equal(t, "10.0.0.1 10.0.0.2 10.0.0.3 10.0.0.4", groups["@all"])
}
