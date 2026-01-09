package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// capsCache cache for device capabilities (key: ip (string) -> DeviceCaps)
var capsCache sync.Map

// sendRPC can be overridden in tests to avoid real network calls
var sendRPC = sendRPCCommand
var rpcPort = yeelightPort
var dialFunc = net.DialTimeout

// sendRPCCommand sends an RPC command to a Yeelight device
//
// Parameters:
//   - ip: device IP address
//   - method: RPC method
//   - params: command parameters
//   - needResponse: flag indicating if response is needed
//
// Returns:
//   - map[string]interface{}: response from device
//   - error: error during command execution
func sendRPCCommand(ip, method string, params []interface{}, needResponse bool) (map[string]interface{}, error) {
	msg := map[string]interface{}{
		"id":     1,
		"method": method,
		"params": params,
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}
	b = append(b, '\r', '\n')

	address := net.JoinHostPort(ip, rpcPort)
	conn, err := dialFunc("tcp", address, connectionTimeout)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	timeout := readTimeout
	if !needResponse {
		timeout = sendTimeout
	}
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return nil, fmt.Errorf("failed to set deadline: %w", err)
	}

	if _, err := conn.Write(b); err != nil {
		return nil, fmt.Errorf("write failed: %w", err)
	}

	if !needResponse {
		return nil, nil
	}

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}

	var res map[string]interface{}
	if err := json.Unmarshal(buf[:n], &res); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return res, nil
}

// detectCaps determines device capabilities
//
// Parameters:
//   - ip: device IP address
//
// Returns:
//   - DeviceCaps: structure with device capabilities
func detectCaps(ip string) DeviceCaps {
	if v, ok := capsCache.Load(ip); ok {
		return v.(DeviceCaps)
	}
	caps := DeviceCaps{HasBG: false}
	if res, err := sendRPC(ip, "get_prop", []interface{}{"bg_power"}, true); err == nil {
		if arr, ok := res["result"].([]interface{}); ok && len(arr) == 1 {
			if s, ok := arr[0].(string); ok && (s == "on" || s == "off") {
				caps.HasBG = true
			}
		}
	}
	capsCache.Store(ip, caps)
	return caps
}

// sendCommand sends a command to a device (without waiting for response)
//
// Parameters:
//   - ip: device IP address
//   - method: RPC method
//   - params: command parameters
func sendCommand(ip, method string, params []interface{}) {
	_, err := sendRPC(ip, method, params, false)
	if err != nil {
		log.Printf("[%s] Error sending command %s: %v", ip, method, err)
	}
}

// sendByTarget sends a command to a specified channel (Main, BG or Both)
//
// Parameters:
//   - ip: device IP address
//   - target: target channel
//   - baseMethod: base RPC method
//   - params: command parameters
func sendByTarget(ip string, target Target, baseMethod string, params []interface{}) {
	c := detectCaps(ip)
	switch target {
	case Main:
		sendCommand(ip, baseMethod, params)
	case BG:
		if c.HasBG {
			sendCommand(ip, "bg_"+baseMethod, params)
		} else {
			log.Printf("[%s] BG command skipped: no ambient channel", ip)
		}
	case Both:
		sendCommand(ip, baseMethod, params)
		if c.HasBG {
			sendCommand(ip, "bg_"+baseMethod, params)
		}
	}
}
