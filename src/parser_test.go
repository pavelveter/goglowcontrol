package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseIPRange(t *testing.T) {
	ips := parseIPRange("192.168.1.1-3")
	require.Equal(t, []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}, ips)
	require.Nil(t, parseIPRange("192.168.1.10-5"))
	require.Nil(t, parseIPRange("192.168.1.a-5"))
}

func TestIsValidIP(t *testing.T) {
	require.True(t, isValidIP("10.0.0.1"))
	require.False(t, isValidIP("10.0.0"))
	require.False(t, isValidIP("abc.def.1.2"))
}

func TestAddArgToIPs(t *testing.T) {
	var ips []string
	cmd, param := "", ""

	addArgToIPs("10.0.0.1-2", &ips, &cmd, &param)
	require.Equal(t, []string{"10.0.0.1", "10.0.0.2"}, ips)
	require.Empty(t, cmd)

	addArgToIPs("on", &ips, &cmd, &param)
	require.Equal(t, "on", cmd)

	addArgToIPs("blue", &ips, &cmd, &param)
	require.Equal(t, "blue", param)

	addArgToIPs("10.0.0.3", &ips, &cmd, &param)
	require.Equal(t, []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"}, ips)
}

func TestParseTargetAndAction(t *testing.T) {
	target, action := parseTargetAndAction("bg-on")
	require.Equal(t, BG, target)
	require.Equal(t, "on", action)

	target, action = parseTargetAndAction("both-color")
	require.Equal(t, Both, target)
	require.Equal(t, "color", action)

	target, action = parseTargetAndAction(" T ")
	require.Equal(t, Main, target)
	require.Equal(t, "t", action)
}
