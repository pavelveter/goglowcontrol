package main

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type rpcCall struct {
	ip           string
	method       string
	params       []interface{}
	needResponse bool
}

func withRPCTestStub(t *testing.T, stub func(ip, method string, params []interface{}, needResponse bool) (map[string]interface{}, error)) {
	orig := sendRPC
	sendRPC = stub
	t.Cleanup(func() { sendRPC = orig })
}

func TestSendRPCCommandRoundTrip(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})

	origDial := dialFunc
	dialFunc = func(network, address string, timeout time.Duration) (net.Conn, error) {
		return client, nil
	}
	t.Cleanup(func() { dialFunc = origDial })

	go func() {
		defer func() { _ = server.Close() }()
		buf := make([]byte, 512)
		n, _ := server.Read(buf)
		_ = server.SetDeadline(time.Now().Add(time.Second))
		_, _ = server.Write([]byte(`{"result":["ok"]}`))
		if n == 0 {
			return
		}
	}()

	res, err := sendRPCCommand("127.0.0.1", "get_prop", []interface{}{"bg_power"}, true)
	require.NoError(t, err)
	require.Equal(t, map[string]interface{}{"result": []interface{}{"ok"}}, res)
}

func TestDetectCapsCaching(t *testing.T) {
	capsCache = sync.Map{}
	var calls int

	withRPCTestStub(t, func(ip, method string, params []interface{}, needResponse bool) (map[string]interface{}, error) {
		calls++
		return map[string]interface{}{"result": []interface{}{"on"}}, nil
	})

	c1 := detectCaps("10.0.0.1")
	c2 := detectCaps("10.0.0.1")

	require.True(t, c1.HasBG)
	require.True(t, c2.HasBG)
	require.Equal(t, 1, calls, "capabilities must be cached per IP")
}

func TestSendByTargetRespectsCaps(t *testing.T) {
	capsCache = sync.Map{}
	capsCache.Store("10.0.0.1", DeviceCaps{HasBG: true})
	capsCache.Store("10.0.0.2", DeviceCaps{HasBG: false})

	var calls []rpcCall
	withRPCTestStub(t, func(ip, method string, params []interface{}, needResponse bool) (map[string]interface{}, error) {
		calls = append(calls, rpcCall{ip: ip, method: method, params: params, needResponse: needResponse})
		return nil, nil
	})

	sendByTarget("10.0.0.1", Main, "set_power", []interface{}{"on"})
	sendByTarget("10.0.0.1", BG, "set_power", []interface{}{"on"})
	sendByTarget("10.0.0.1", Both, "set_power", []interface{}{"on"})
	sendByTarget("10.0.0.2", BG, "set_power", []interface{}{"on"})

	require.Equal(t, 4, len(calls))
	require.Equal(t, []string{"set_power", "bg_set_power", "set_power", "bg_set_power"}, []string{
		calls[0].method, calls[1].method, calls[2].method, calls[3].method,
	})
	for _, c := range calls[:3] {
		require.False(t, c.needResponse)
		require.Equal(t, []interface{}{"on"}, c.params)
	}
}

func TestExecuteCommandDispatchesActions(t *testing.T) {
	capsCache = sync.Map{}
	capsCache.Store("10.0.0.1", DeviceCaps{HasBG: true})

	colorMap.mu.Lock()
	colorMap.colors = map[string]int64{
		"red":  0xFF0000,
		"blue": 0x0000FF,
	}
	colorMap.mu.Unlock()

	var calls []rpcCall
	withRPCTestStub(t, func(ip, method string, params []interface{}, needResponse bool) (map[string]interface{}, error) {
		calls = append(calls, rpcCall{ip: ip, method: method, params: params, needResponse: needResponse})
		return nil, nil
	})

	executeCommand("10.0.0.1", "color", "red")
	executeCommand("10.0.0.1", "brightness", "15")
	executeCommand("10.0.0.1", "bg-on", "")
	executeCommand("10.0.0.1", "both-undim", "")
	executeCommand("10.0.0.1", "notify-blue", "")

	require.Len(t, calls, 6)
	require.Equal(t, "set_scene", calls[0].method)
	require.Equal(t, []interface{}{"color", int64(0xFF0000), maxBrightness}, calls[0].params)

	require.Equal(t, "set_bright", calls[1].method)
	require.Equal(t, []interface{}{15}, calls[1].params)

	require.Equal(t, "bg_set_power", calls[2].method)
	require.Equal(t, []interface{}{"on", "smooth", defaultSmoothDuration}, calls[2].params)

	require.Equal(t, "set_bright", calls[3].method)
	require.Equal(t, []interface{}{defaultBrightValue}, calls[3].params)
	require.Equal(t, "bg_set_bright", calls[4].method)
	require.Equal(t, []interface{}{defaultBrightValue}, calls[4].params)

	require.Equal(t, "start_cf", calls[5].method)
	require.Equal(t, "100, 1, 255, 100, 100, 1, 255, 1", calls[5].params[2])
}

func TestExecuteCommandValidationErrors(t *testing.T) {
	var calls []rpcCall
	withRPCTestStub(t, func(ip, method string, params []interface{}, needResponse bool) (map[string]interface{}, error) {
		calls = append(calls, rpcCall{})
		return nil, nil
	})

	executeCommand("10.0.0.1", "brightness", "200") // invalid
	executeCommand("10.0.0.1", "t", "1000")         // invalid
	executeCommand("10.0.0.1", "unknown", "")

	require.Empty(t, calls, "invalid commands should not call RPC")
}

func TestExecuteSceneResolvesAliasesAndRanges(t *testing.T) {
	origScenes := scenes
	origBulbGroups := bulbGroups
	t.Cleanup(func() {
		scenes = origScenes
		bulbGroups = origBulbGroups
	})

	scenes = map[string]Scene{
		"combo": {
			Name: "combo",
			Commands: []string{
				"10.0.0.1 on",
				"@room color red",
				"192.168.0.1-2 both-brightness 10",
			},
		},
	}
	bulbGroups = map[string]string{
		"@room": "10.0.0.2 10.0.0.3",
	}
	capsCache = sync.Map{}
	for _, ip := range []string{"10.0.0.1", "10.0.0.2", "10.0.0.3", "192.168.0.1", "192.168.0.2"} {
		capsCache.Store(ip, DeviceCaps{HasBG: true})
	}

	colorMap.mu.Lock()
	colorMap.colors = map[string]int64{"red": 0xFF0000}
	colorMap.mu.Unlock()

	var calls []rpcCall
	withRPCTestStub(t, func(ip, method string, params []interface{}, needResponse bool) (map[string]interface{}, error) {
		calls = append(calls, rpcCall{ip: ip, method: method, params: params, needResponse: needResponse})
		return nil, nil
	})

	executeScene("combo")

	require.Len(t, calls, 7)
	require.Equal(t, []string{
		"set_power",
		"set_scene",
		"set_scene",
		"set_bright", "bg_set_bright",
		"set_bright", "bg_set_bright",
	}, []string{
		calls[0].method, calls[1].method, calls[2].method,
		calls[3].method, calls[4].method, calls[5].method, calls[6].method,
	})
}

func TestMainRunsCommand(t *testing.T) {
	origArgs := os.Args
	os.Args = []string{"light", "10.0.0.1", "on"}
	defer func() { os.Args = origArgs }()

	capsCache = sync.Map{}
	capsCache.Store("10.0.0.1", DeviceCaps{HasBG: true})

	var (
		mu    sync.Mutex
		calls []rpcCall
	)
	withRPCTestStub(t, func(ip, method string, params []interface{}, needResponse bool) (map[string]interface{}, error) {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, rpcCall{ip: ip, method: method, params: params, needResponse: needResponse})
		return nil, nil
	})

	main()

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, calls, 1)
	require.Equal(t, "10.0.0.1", calls[0].ip)
	require.Equal(t, "set_power", calls[0].method)
	require.Equal(t, []interface{}{"on", "smooth", defaultSmoothDuration}, calls[0].params)
	require.False(t, calls[0].needResponse)
}

func TestSendCommandLogsError(t *testing.T) {
	orig := sendRPC
	defer func() { sendRPC = orig }()

	sendRPC = func(ip, method string, params []interface{}, needResponse bool) (map[string]interface{}, error) {
		return nil, fmt.Errorf("boom")
	}

	sendCommand("10.0.0.1", "set_power", []interface{}{"on"})
}

func TestMainHandlesAliasAndNumericBrightness(t *testing.T) {
	origArgs := os.Args
	os.Args = []string{"light", "@room", "30"}
	defer func() { os.Args = origArgs }()

	origBulbGroups := bulbGroups
	bulbGroups = map[string]string{"@room": "10.0.0.1 10.0.0.2"}
	t.Cleanup(func() { bulbGroups = origBulbGroups })

	capsCache = sync.Map{}
	capsCache.Store("10.0.0.1", DeviceCaps{HasBG: true})
	capsCache.Store("10.0.0.2", DeviceCaps{HasBG: true})

	var mu sync.Mutex
	var calls []rpcCall
	withRPCTestStub(t, func(ip, method string, params []interface{}, needResponse bool) (map[string]interface{}, error) {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, rpcCall{ip: ip, method: method, params: params, needResponse: needResponse})
		return nil, nil
	})

	main()

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, calls, 2)
	require.ElementsMatch(t, []string{"10.0.0.1", "10.0.0.2"}, []string{calls[0].ip, calls[1].ip})
	for _, c := range calls {
		require.Equal(t, "set_bright", c.method)
		require.Equal(t, []interface{}{30}, c.params)
	}
}

func TestMainShowsHelpWithoutExit(t *testing.T) {
	origArgs := os.Args
	os.Args = []string{"light"}
	defer func() { os.Args = origArgs }()

	origExit := exitFunc
	exitCalled := false
	exitFunc = func(code int) { exitCalled = true }
	t.Cleanup(func() { exitFunc = origExit })

	origStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	main()

	require.NoError(t, w.Close())
	os.Stdout = origStdout

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err)
	require.True(t, exitCalled)
	require.Contains(t, buf.String(), "Usage:")
}
