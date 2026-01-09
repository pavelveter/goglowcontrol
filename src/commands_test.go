package main

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExecuteCommandVariousActions(t *testing.T) {
	capsCache = sync.Map{}
	capsCache.Store("10.0.0.1", DeviceCaps{HasBG: true})

	colorMap.mu.Lock()
	colorMap.colors = map[string]int64{
		"red":   0xFF0000,
		"white": 0xFFFFFF,
	}
	colorMap.mu.Unlock()

	var calls []rpcCall
	withRPCTestStub(t, func(ip, method string, params []interface{}, needResponse bool) (map[string]interface{}, error) {
		calls = append(calls, rpcCall{ip: ip, method: method, params: params, needResponse: needResponse})
		return nil, nil
	})

	executeCommand("10.0.0.1", "t", "2000")
	executeCommand("10.0.0.1", "disco", "")
	executeCommand("10.0.0.1", "sunrise", "")
	executeCommand("10.0.0.1", "dim", "")
	executeCommand("10.0.0.1", "undim", "")
	executeCommand("10.0.0.1", "50", "")

	require.GreaterOrEqual(t, len(calls), 6)
	require.Equal(t, "set_ct_abx", calls[0].method)
	require.Equal(t, []interface{}{2000, "smooth", defaultSmoothDuration}, calls[0].params)
	require.Equal(t, "start_cf", calls[1].method)
	require.Equal(t, "start_cf", calls[2].method)
	require.Equal(t, "set_bright", calls[3].method)
	require.Equal(t, "set_bright", calls[4].method)
	require.Equal(t, "set_bright", calls[5].method)
}
