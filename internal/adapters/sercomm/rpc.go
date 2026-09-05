package sercomm

import (
	"encoding/json"
	"fmt"
)

// RPCRequest is a JSON-RPC 2.0 read request for /data/data.cgi.
type RPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  string `json:"params"`
	ID      int    `json:"id"`
}

// RPCResponse is a JSON-RPC 2.0 response from /data/data.cgi.
type RPCResponse struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *RPCError       `json:"error,omitempty"`
}

// RPCError represents an error block in a JSON-RPC 2.0 response.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// HostItem represents one device in the "hosts.@host" array.
type HostItem struct {
	Hostname string `json:"hostname"`
	HostIP   string `json:"hostip"`
	HostMAC  string `json:"hostmac"`
	HostIPv6 string `json:"hostipv6"`
	ConnType string `json:"conntype"`
	Alive    string `json:"alive"`
}

// WirelessGeneral represents the "wireless.general" result object.
type WirelessGeneral struct {
	WPS      string `json:"wps"`
	Schedule string `json:"schedule"`
}

// WANInfo represents the "scinfo.@wan" item.
type WANInfo struct {
	Status string `json:"status"`
	Type   string `json:"type"`
}

// ParseHostsResult extracts the HostItem list from a JSON-RPC response
// corresponding to "hosts.@host".
func ParseHostsResult(raw json.RawMessage) ([]HostItem, error) {
	var container map[string]json.RawMessage
	if err := json.Unmarshal(raw, &container); err != nil {
		return nil, fmt.Errorf("unmarshal hosts result container: %w", err)
	}
	rawHosts, ok := container["hosts.@host"]
	if !ok {
		return nil, nil
	}
	var items []HostItem
	if err := json.Unmarshal(rawHosts, &items); err != nil {
		return nil, fmt.Errorf("unmarshal hosts items: %w", err)
	}
	return items, nil
}

// ParseWirelessResult extracts the WirelessGeneral state from a JSON-RPC response.
func ParseWirelessResult(raw json.RawMessage) (WirelessGeneral, error) {
	var container map[string]json.RawMessage
	if err := json.Unmarshal(raw, &container); err != nil {
		return WirelessGeneral{}, fmt.Errorf("unmarshal wireless result container: %w", err)
	}
	rawGen, ok := container["wireless.general"]
	if !ok {
		return WirelessGeneral{}, fmt.Errorf("wireless.general key missing")
	}
	var gen WirelessGeneral
	if err := json.Unmarshal(rawGen, &gen); err != nil {
		return WirelessGeneral{}, fmt.Errorf("unmarshal wireless general: %w", err)
	}
	return gen, nil
}
