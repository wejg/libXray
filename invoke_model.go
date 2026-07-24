// libXray is an Xray wrapper focusing on improving the experience of Xray-core mobile development.
package libXray

import "encoding/json"

type LibXrayMethod string

const (
	LibXrayMethodGetFreePorts                LibXrayMethod = "getFreePorts"
	LibXrayMethodConvertShareLinksToXrayJson LibXrayMethod = "convertShareLinksToXrayJson"
	LibXrayMethodConvertXrayJsonToShareLinks LibXrayMethod = "convertXrayJsonToShareLinks"
	LibXrayMethodCountGeoData                LibXrayMethod = "countGeoData"
	LibXrayMethodReadGeoFiles                LibXrayMethod = "readGeoFiles"
	LibXrayMethodPing                        LibXrayMethod = "ping"
	LibXrayMethodQueryStats                  LibXrayMethod = "queryStats"
	LibXrayMethodQueryStatsByTag             LibXrayMethod = "queryStatsByTag"
	LibXrayMethodTestXray                    LibXrayMethod = "testXray"
	LibXrayMethodRunXray                     LibXrayMethod = "runXray"
	LibXrayMethodRunXrayFromJson             LibXrayMethod = "runXrayFromJson"
	LibXrayMethodStopXray                    LibXrayMethod = "stopXray"
	LibXrayMethodXrayVersion                 LibXrayMethod = "xrayVersion"
	LibXrayMethodGetXrayState                LibXrayMethod = "getXrayState"
	LibXrayMethodSetMemoryLimit              LibXrayMethod = "setMemoryLimit"
	LibXrayMethodReplaceInbound              LibXrayMethod = "replaceInbound"
	LibXrayMethodReplaceOutbound             LibXrayMethod = "replaceOutbound"
	LibXrayMethodAddInbound                  LibXrayMethod = "addInbound"
	LibXrayMethodRemoveInbound               LibXrayMethod = "removeInbound"
	LibXrayMethodAddOutbound                 LibXrayMethod = "addOutbound"
	LibXrayMethodRemoveOutbound              LibXrayMethod = "removeOutbound"
	LibXrayMethodAddRouteRule                LibXrayMethod = "addRouteRule"
	LibXrayMethodRemoveRouteRule             LibXrayMethod = "removeRouteRule"
)

type LibXrayInvokeRequest struct {
	APIVersion int             `json:"apiVersion,omitempty"`
	Method     LibXrayMethod   `json:"method,omitempty"`
	Payload    json.RawMessage `json:"payload,omitempty"`
}

type GetFreePortsRequest struct {
	Count int `json:"count,omitempty"`
}

type GetFreePortsResponse struct {
	Ports []int `json:"ports,omitempty"`
}

type ConvertShareLinksToXrayJsonRequest struct {
	Text string `json:"text,omitempty"`
}

type ConvertXrayJsonToShareLinksRequest struct {
	XrayJson string `json:"xrayJson,omitempty"`
}

type ConvertXrayJsonToShareLinksResponse struct {
	Links string `json:"links,omitempty"`
}

type CountGeoDataRequest struct {
	Name    string `json:"name,omitempty"`
	GeoType string `json:"geoType,omitempty"`
	DatDir  string `json:"datDir,omitempty"`
}

type ReadGeoFilesRequest struct {
	ConfigJSON string `json:"configJSON,omitempty"`
}

type ReadGeoFilesResponse struct {
	Domain []string `json:"domain,omitempty"`
	IP     []string `json:"ip,omitempty"`
}

type PingRequest struct {
	ConfigPath string `json:"configPath,omitempty"`
	Timeout    int    `json:"timeout,omitempty"`
	URL        string `json:"url,omitempty"`
	Proxy      string `json:"proxy,omitempty"`
}

type PingResponse struct {
	Delay int64 `json:"delay,omitempty"`
}

type QueryStatsRequest struct {
	Server string `json:"server,omitempty"`
}

type QueryStatsByTagRequest struct {
	Tag string `json:"tag,omitempty"`
}

type QueryStatsResponse struct {
	Stats string `json:"stats,omitempty"`
}

type RunXrayRequest struct {
	ConfigPath string `json:"configPath,omitempty"`
}

type RunXrayFromJSONRequest struct {
	ConfigJSON string `json:"configJSON,omitempty"`
}

type XrayVersionResponse struct {
	Version string `json:"version,omitempty"`
}

type GetXrayStateResponse struct {
	Running bool `json:"running"`
}

type SetMemoryLimitRequest struct {
	MemoryMB int `json:"memoryMB,omitempty"`
}

type InboundJSONRequest struct {
	InboundJSON string `json:"inboundJSON,omitempty"`
}

type OutboundJSONRequest struct {
	OutboundJSON string `json:"outboundJSON,omitempty"`
}

type TagRequest struct {
	Tag string `json:"tag,omitempty"`
}

type AddRouteRuleRequest struct {
	Rule         string `json:"rule,omitempty"`
	ShouldAppend bool   `json:"shouldAppend,omitempty"`
}

type RemoveRouteRuleRequest struct {
	RuleTag string `json:"ruleTag,omitempty"`
}
