package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSupportsColorRespectsEnv(t *testing.T) {
	orig := os.Getenv("NO_COLOR")
	t.Cleanup(func() { _ = os.Setenv("NO_COLOR", orig) })

	_ = os.Setenv("NO_COLOR", "1")
	require.False(t, supportsColor())
}

func TestPrintWrappedListOutputsItems(t *testing.T) {
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	printWrappedList("Title", []string{"alpha", "beta", "gamma", "delta"}, 20)

	require.NoError(t, w.Close())
	os.Stdout = origStdout

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	output := buf.String()
	require.Contains(t, output, "Title")
	require.True(t, strings.Contains(output, "alpha") || strings.Contains(output, "beta"))
}

func TestPrintWrappedListColorizedNoPanic(t *testing.T) {
	origUseColors := useColors
	useColors = false
	t.Cleanup(func() { useColors = origUseColors })

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	printWrappedListColorized("Colors", []string{"red", "green", "blue"}, 30)

	require.NoError(t, w.Close())
	os.Stdout = origStdout

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	require.Contains(t, buf.String(), "Colors")
}

func TestGetLocalNetworkPrefixDoesNotPanic(t *testing.T) {
	prefix := getLocalNetworkPrefix()
	if prefix != "" {
		require.Contains(t, prefix, ".")
	}
}

func TestPrintHelpDoesNotExitDuringTest(t *testing.T) {
	origExit := exitFunc
	exitCalled := false
	exitFunc = func(code int) { exitCalled = true }
	t.Cleanup(func() { exitFunc = origExit })

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	os.Args = []string{"light"}
	printHelp()

	require.NoError(t, w.Close())
	os.Stdout = origStdout

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	require.True(t, exitCalled)
	require.Contains(t, buf.String(), "Usage:")
}
