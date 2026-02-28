package xray

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	corestats "github.com/xtls/xray-core/features/stats"
)

// query inbound and outbound stats.
// server is either:
//   - a metrics URL (e.g. "http://[::1]:49227/debug/vars") → HTTP GET, returns expvar body;
//   - an outbound tag (e.g. "proxy") → in-process stats via corestats.Manager, no metrics needed.
func QueryStats(server string) (string, error) {
	// If it looks like a URL, use HTTP (legacy metrics).
	if strings.Contains(server, "://") {
		return queryStatsHTTP(server)
	}
	// Otherwise treat as outbound tag and read from running core (no metrics, no expvar crash).
	return queryStatsInProcess(strings.TrimSpace(server))
}

// QueryStatsByTag returns uplink/downlink for the given outbound tag via in-process corestats only.
// Does not use any URL or metrics; safe when metrics is disabled to avoid expvar crash.
func QueryStatsByTag(tag string) (string, error) {
	return queryStatsInProcess(strings.TrimSpace(tag))
}

func queryStatsHTTP(server string) (string, error) {
	resp, err := http.Get(server)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// queryStatsInProcess reads uplink/downlink from corestats.Manager (same as AndroidLibXrayLite).
// Counter names: outbound>>>tag>>>traffic>>>uplink / downlink. Value() returns cumulative bytes.
func queryStatsInProcess(tag string) (string, error) {
	if tag == "" {
		return "", fmt.Errorf("stats tag is empty")
	}
	if coreServer == nil || !coreServer.IsRunning() {
		return "", fmt.Errorf("core not running")
	}
	m := coreServer.GetFeature(corestats.ManagerType())
	if m == nil {
		return "", fmt.Errorf("stats not enabled in config")
	}
	manager, ok := m.(corestats.Manager)
	if !ok {
		return "", fmt.Errorf("stats manager type mismatch")
	}
	upCounter := manager.GetCounter(fmt.Sprintf("outbound>>>%s>>>traffic>>>uplink", tag))
	downCounter := manager.GetCounter(fmt.Sprintf("outbound>>>%s>>>traffic>>>downlink", tag))
	var uplink, downlink int64
	if upCounter != nil {
		uplink = upCounter.Value()
	}
	if downCounter != nil {
		downlink = downCounter.Value()
	}
	out := struct {
		Uplink   int64 `json:"uplink"`
		Downlink int64 `json:"downlink"`
	}{Uplink: uplink, Downlink: downlink}
	raw, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
