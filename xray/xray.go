package xray

import (
	"context"
	"encoding/json"
	"os"
	"runtime/debug"
	"strconv"
	"sync"

	"github.com/xtls/libxray/memory"
	"github.com/xtls/libxray/nodep"
	"github.com/xtls/xray-core/common/cmdarg"
	"github.com/xtls/xray-core/common/errors"
	"github.com/xtls/xray-core/common/platform"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/core"

	"github.com/xtls/xray-core/features/inbound"
	"github.com/xtls/xray-core/features/outbound"
	"github.com/xtls/xray-core/features/routing"
	"github.com/xtls/xray-core/infra/conf"
	_ "github.com/xtls/xray-core/main/distro/all"
)

var (
	coreServer   *core.Instance
	coreServerMu sync.RWMutex
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
	coreServerMu.Lock()
	defer coreServerMu.Unlock()

	InitEnv(datDir)
	memory.InitForceFree()
	server, err := StartXray(configPath)
	if err != nil {
		return
	}

	if err = server.Start(); err != nil {
		return
	}
	coreServer = server

	debug.FreeOSMemory()
	return nil
}

// Run Xray instance with JSON configuration string.
// datDir means the dir which geosite.dat and geoip.dat are in.
// configJSON means the JSON configuration string.
func RunXrayFromJSON(datDir, configJSON string) (err error) {
	coreServerMu.Lock()
	defer coreServerMu.Unlock()

	InitEnv(datDir)
	memory.InitForceFree()
	server, err := StartXrayFromJSON(configJSON)
	if err != nil {
		return
	}
	coreServer = server

	debug.FreeOSMemory()
	return nil
}

// Get Xray State
func GetXrayState() bool {
	coreServerMu.RLock()
	defer coreServerMu.RUnlock()
	return coreServer != nil && coreServer.IsRunning()
}

// Stop Xray instance.
func StopXray() error {
	coreServerMu.Lock()
	defer coreServerMu.Unlock()
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
	coreServerMu.RLock()
	defer coreServerMu.RUnlock()
	server := coreServer
	if server == nil || !server.IsRunning() {
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
	om := server.GetFeature(outbound.ManagerType()).(outbound.Manager)
	_ = om.RemoveHandler(context.Background(), od.Tag)
	return core.AddOutboundHandler(server, handlerCfg)
}

// ReplaceInbound hot-swaps the inbound whose tag matches the one in inboundJSON,
// without restarting the core.
// inboundJSON is a single inbound object (the same structure as one item in the "inbounds" array).
func ReplaceInbound(inboundJSON string) error {
	coreServerMu.RLock()
	defer coreServerMu.RUnlock()
	server := coreServer
	if server == nil || !server.IsRunning() {
		return errors.New("xray not running")
	}
	var id conf.InboundDetourConfig
	if err := json.Unmarshal([]byte(inboundJSON), &id); err != nil {
		return err
	}
	handlerCfg, err := id.Build()
	if err != nil {
		return err
	}
	im := server.GetFeature(inbound.ManagerType()).(inbound.Manager)
	_ = im.RemoveHandler(context.Background(), id.Tag)
	return core.AddInboundHandler(server, handlerCfg)
}

// AddRouteRule hot-adds a single routing rule without restarting the core.
// shouldAppend=true appends the rule to the end of the existing rule list.
// shouldAppend=false REPLACES the entire rule list with just this rule (the core
// has no "insert at front" capability; rules can only be appended). For hot-adding
// a rule you almost always want shouldAppend=true.
// ruleJSON is a single rule object (the same structure as one item in
// "routing.rules"). To later remove it via RemoveRouteRule, include a "ruleTag" field.
func AddRouteRule(ruleJSON string, shouldAppend bool) error {
	coreServerMu.RLock()
	defer coreServerMu.RUnlock()
	server := coreServer
	if server == nil || !server.IsRunning() {
		return errors.New("xray not running")
	}
	// Reuse the exported RouterConfig.Build to turn the single rule JSON into a
	// *router.Config (parseRule itself is unexported). Router.AddRule expects the
	// TypedMessage to carry a *router.Config, then reloads the rules from it.
	rc := conf.RouterConfig{RuleList: []json.RawMessage{json.RawMessage(ruleJSON)}}
	built, err := rc.Build()
	if err != nil {
		return err
	}
	if len(built.Rule) == 0 {
		return errors.New("no rule built from json")
	}
	r := server.GetFeature(routing.RouterType()).(routing.Router)
	return r.AddRule(serial.ToTypedMessage(built), shouldAppend)
}

// RemoveRouteRule removes a routing rule by its ruleTag.
func RemoveRouteRule(ruleTag string) error {
	coreServerMu.RLock()
	defer coreServerMu.RUnlock()
	server := coreServer
	if server == nil || !server.IsRunning() {
		return errors.New("xray not running")
	}
	r := server.GetFeature(routing.RouterType()).(routing.Router)
	return r.RemoveRule(ruleTag)
}

// AddInbound hot-adds a single inbound without restarting the core.
// inboundJSON is a single inbound object (the same structure as one item in the
// "inbounds" array); its tag is read from the JSON. The tag must be unique — if an
// inbound with the same tag already exists the core returns "existing tag found".
// Use ReplaceInbound when you want replace-by-tag semantics instead.
func AddInbound(inboundJSON string) error {
	coreServerMu.RLock()
	defer coreServerMu.RUnlock()
	server := coreServer
	if server == nil || !server.IsRunning() {
		return errors.New("xray not running")
	}
	var id conf.InboundDetourConfig
	if err := json.Unmarshal([]byte(inboundJSON), &id); err != nil {
		return err
	}
	handlerCfg, err := id.Build()
	if err != nil {
		return err
	}
	return core.AddInboundHandler(server, handlerCfg)
}

// RemoveInbound removes a running inbound by its tag, closing its listener.
func RemoveInbound(tag string) error {
	coreServerMu.RLock()
	defer coreServerMu.RUnlock()
	server := coreServer
	if server == nil || !server.IsRunning() {
		return errors.New("xray not running")
	}
	im := server.GetFeature(inbound.ManagerType()).(inbound.Manager)
	return im.RemoveHandler(context.Background(), tag)
}

// AddOutbound hot-adds a single outbound without restarting the core.
// outboundJSON is a single outbound object (the same structure as one item in the
// "outbounds" array); its tag is read from the JSON. The tag must be unique — if an
// outbound with the same tag already exists the core returns "existing tag found".
// Use ReplaceOutbound when you want replace-by-tag semantics instead.
func AddOutbound(outboundJSON string) error {
	coreServerMu.RLock()
	defer coreServerMu.RUnlock()
	server := coreServer
	if server == nil || !server.IsRunning() {
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
	return core.AddOutboundHandler(server, handlerCfg)
}

// RemoveOutbound removes a running outbound by its tag.
func RemoveOutbound(tag string) error {
	coreServerMu.RLock()
	defer coreServerMu.RUnlock()
	server := coreServer
	if server == nil || !server.IsRunning() {
		return errors.New("xray not running")
	}
	om := server.GetFeature(outbound.ManagerType()).(outbound.Manager)
	return om.RemoveHandler(context.Background(), tag)
}

// Xray's version
func XrayVersion() string {
	return core.Version()
}
