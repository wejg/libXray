package xray

import (
	"context"
	"math"
	"net"
	"net/http"
	"time"

	"github.com/xtls/libxray/nodep"
	xnet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/core"
)

// Ping Xray config and find the delay and country code of its outbound.
// datDir means the dir which geosite.dat and geoip.dat are in.
// configPath means the config.json file path.
// timeout means how long the http request will be cancelled if no response, in units of seconds.
// url means the website we use to test speed. "https://www.google.com" is a good choice for most cases.
// proxy means the local http/socks5 proxy, like "socks5://[::1]:1080".
func Ping(datDir string, configPath string, timeout int, url string, proxy string) (int64, error) {
	if coreServer != nil && coreServer.IsRunning() {
		return measureDelayWithRunningCore(timeout, url)
	}

	InitEnv(datDir)
	server, err := StartXray(configPath)
	if err != nil {
		return nodep.PingDelayError, err
	}

	if err := server.Start(); err != nil {
		return nodep.PingDelayError, err
	}
	defer server.Close()

	delay, err := nodep.MeasureDelay(timeout, url, proxy)
	if err != nil {
		return delay, err
	}

	return delay, nil
}

func measureDelayWithRunningCore(timeout int, url string) (int64, error) {
	httpTimeout := time.Second * time.Duration(timeout)
	client := &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
			DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				dest, err := xnet.ParseDestination(network + ":" + address)
				if err != nil {
					return nil, err
				}
				if dest.Network == xnet.Network_Unknown {
					dest.Network = xnet.Network_TCP
				}
				return core.Dial(ctx, coreServer, dest)
			},
		},
		Timeout: httpTimeout,
	}

	start := time.Now()
	req, _ := http.NewRequest("HEAD", url, nil)
	_, err := client.Do(req)
	delay := time.Since(start).Milliseconds()
	if err != nil {
		precision := delay - int64(timeout)*1000
		if math.Abs(float64(precision)) < 50 {
			return nodep.PingDelayTimeout, err
		}
		return nodep.PingDelayError, err
	}
	return delay, nil
}
