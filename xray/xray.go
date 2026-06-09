package xray

import (
	"context"
	"encoding/json"
	"os"
	"runtime/debug"
	"strconv"

	"github.com/xtls/libxray/memory"
	"github.com/xtls/libxray/nodep"
	"github.com/xtls/xray-core/common/cmdarg"
	"github.com/xtls/xray-core/common/errors"
	"github.com/xtls/xray-core/common/platform"
	"github.com/xtls/xray-core/core"

	"github.com/xtls/xray-core/features/outbound"
	"github.com/xtls/xray-core/infra/conf"
	_ "github.com/xtls/xray-core/main/distro/all"
)

var (
	coreServer *core.Instance
)

func StartXray(configPath string) (*core.Instance, error) {
	file := cmdarg.Arg{configPath}
	config, err := core.LoadConfig("json", file)
	if err != nil {
		return nil, err
	}

	server, err := core.New(config)
	if err != nil {
		return nil, err
	}

	return server, nil
}

func StartXrayFromJSON(configJSON string) (*core.Instance, error) {
	// Convert JSON string to bytes
	configBytes := []byte(configJSON)

	// Use core.StartInstance which can load configuration directly from bytes
	server, err := core.StartInstance("json", configBytes)
	if err != nil {
		return nil, err
	}

	return server, nil
}

// SetTunFd sets the TUN file descriptor.
// Call this BEFORE RunXray/RunXrayFromJSON.
func SetTunFd(fd int32) string {
	err := os.Setenv(platform.TunFdKey, strconv.Itoa(int(fd)))
	var response nodep.CallResponse[string]
	return response.EncodeToBase64("", err)
}

func InitEnv(datDir string) {
	os.Setenv(platform.AssetLocation, datDir)
	os.Setenv(platform.CertLocation, datDir)
}

// Run Xray instance.
// datDir means the dir which geosite.dat and geoip.dat are in.
// configPath means the config.json file path.
func RunXray(datDir, configPath string) (err error) {
	InitEnv(datDir)
	memory.InitForceFree()
	coreServer, err = StartXray(configPath)
	if err != nil {
		return
	}

	if err = coreServer.Start(); err != nil {
		return
	}

	debug.FreeOSMemory()
	return nil
}

// Run Xray instance with JSON configuration string.
// datDir means the dir which geosite.dat and geoip.dat are in.
// configJSON means the JSON configuration string.
func RunXrayFromJSON(datDir, configJSON string) (err error) {
	InitEnv(datDir)
	memory.InitForceFree()
	coreServer, err = StartXrayFromJSON(configJSON)
	if err != nil {
		return
	}

	debug.FreeOSMemory()
	return nil
}

// Get Xray State
func GetXrayState() bool {
	return coreServer != nil && coreServer.IsRunning()
}

// Stop Xray instance.
func StopXray() error {
	if coreServer != nil {
		err := coreServer.Close()
		coreServer = nil
		if err != nil {
			return err
		}
	}
	return nil
}

// ReplaceOutbound hot-swaps the outbound whose tag matches the one in outboundJSON.
// outboundJSON is a single outbound object (the same structure as one item in the "outbounds" array).
func ReplaceOutbound(outboundJSON string) error {
	if coreServer == nil || !coreServer.IsRunning() {
		return errors.New("xray not running")
	}
	var od conf.OutboundDetourConfig
	if err := json.Unmarshal([]byte(outboundJSON), &od); err != nil {
		return err
	}
	handlerCfg, err := od.Build()
	if err != nil {
		return err
	}
	om := coreServer.GetFeature(outbound.ManagerType()).(outbound.Manager)
	_ = om.RemoveHandler(context.Background(), od.Tag)
	return core.AddOutboundHandler(coreServer, handlerCfg)
}

// Xray's version
func XrayVersion() string {
	return core.Version()
}
