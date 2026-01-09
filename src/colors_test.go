package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestColorCacheLoadAndGet(t *testing.T) {
	var cache colorCache
	err := cache.load(`
# comment
Red; FF0000
Green;00ff00
Blue;zzzzzz
`)
	require.NoError(t, err)
	require.Len(t, cache.colors, 2)

	value, ok := cache.get("red")
	require.True(t, ok)
	require.Equal(t, int64(0xFF0000), value)

	value, ok = cache.get("GREEN")
	require.True(t, ok)
	require.Equal(t, int64(0x00FF00), value)

	_, ok = cache.get("blue")
	require.False(t, ok)
}

func TestColorizeAndHexToRGB(t *testing.T) {
	origUseColors := useColors
	t.Cleanup(func() { useColors = origUseColors })

	useColors = true
	require.Equal(t, colorRed+"hi"+colorReset, colorize("hi", colorRed))
	require.Equal(t, "\033[38;2;51;102;153m", hexToRGB(0x336699))

	useColors = false
	require.Equal(t, "hi", colorize("hi", colorRed))
	require.Empty(t, hexToRGB(0x336699))
}
