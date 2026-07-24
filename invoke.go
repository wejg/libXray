package libXray

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/xtls/libxray/geo"
	"github.com/xtls/libxray/memory"
	"github.com/xtls/libxray/nodep"
	"github.com/xtls/libxray/share"
	"github.com/xtls/libxray/xray"
)

type invokeResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data"`
	Err     string `json:"error"`
}

const (
	maxInvokeJSONSizeMiB = 16
	maxInvokeJSONBytes   = maxInvokeJSONSizeMiB * 1024 * 1024
)

func Invoke(requestJSON string) string {
	if len(requestJSON) > maxInvokeJSONBytes {
		return encodeInvokeResponse(nil, errors.New(invokeJSONSizeLimitMessage("request")))
	}
	var request LibXrayInvokeRequest
	if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
		return encodeInvokeResponse(nil, err)
	}
	if err := validateAPIVersion(request.APIVersion); err != nil {
		return encodeInvokeResponse(nil, err)
	}

	switch request.Method {
	case LibXrayMethodGetFreePorts:
		return invokeGetFreePorts(request.Payload)
	case LibXrayMethodConvertShareLinksToXrayJson:
		return invokeConvertShareLinksToXrayJson(request.Payload)
	case LibXrayMethodConvertXrayJsonToShareLinks:
		return invokeConvertXrayJsonToShareLinks(request.Payload)
	case LibXrayMethodCountGeoData:
		return invokeCountGeoData(request.Payload)
	case LibXrayMethodReadGeoFiles:
		return invokeReadGeoFiles(request.Payload)
	case LibXrayMethodPing:
		return invokePing(request.Payload)
	case LibXrayMethodQueryStats:
		return invokeQueryStats(request.Payload)
	case LibXrayMethodQueryStatsByTag:
		return invokeQueryStatsByTag(request.Payload)
	case LibXrayMethodTestXray:
		return invokeTestXray(request.Payload)
	case LibXrayMethodRunXray:
		return invokeRunXray(request.Payload)
	case LibXrayMethodRunXrayFromJson:
		return invokeRunXrayFromJSON(request.Payload)
	case LibXrayMethodStopXray:
		return encodeInvokeNoDataResponse(xray.StopXray())
	case LibXrayMethodXrayVersion:
		return encodeInvokeResponse(&XrayVersionResponse{Version: xray.XrayVersion()}, nil)
	case LibXrayMethodGetXrayState:
		return encodeInvokeResponse(&GetXrayStateResponse{Running: xray.GetXrayState()}, nil)
	case LibXrayMethodSetMemoryLimit:
		return invokeSetMemoryLimit(request.Payload)
	case LibXrayMethodReplaceInbound:
		return invokeReplaceInbound(request.Payload)
	case LibXrayMethodReplaceOutbound:
		return invokeReplaceOutbound(request.Payload)
	case LibXrayMethodAddInbound:
		return invokeAddInbound(request.Payload)
	case LibXrayMethodRemoveInbound:
		return invokeRemoveInbound(request.Payload)
	case LibXrayMethodAddOutbound:
		return invokeAddOutbound(request.Payload)
	case LibXrayMethodRemoveOutbound:
		return invokeRemoveOutbound(request.Payload)
	case LibXrayMethodAddRouteRule:
		return invokeAddRouteRule(request.Payload)
	case LibXrayMethodRemoveRouteRule:
		return invokeRemoveRouteRule(request.Payload)
	default:
		return encodeInvokeResponse(nil, errors.New("unknown method"))
	}
}
func validateAPIVersion(version int) error {
	if version == 0 || version == 1 {
		return nil
	}
	return errors.New("unsupported apiVersion")
}

func decodePayload[T any](payload json.RawMessage) (T, error) {
	var request T
	if len(payload) == 0 {
		return request, nil
	}
	err := json.Unmarshal(payload, &request)
	return request, err
}

func encodeInvokeResponse(data any, err error) string {
	response := invokeResponse{Data: data}
	if err != nil {
		response.Success = false
		response.Err = err.Error()
	} else {
		response.Success = true
	}
	raw, err := json.Marshal(&response)
	if err != nil {
		return `{"success":false,"data":null,"error":"failed to encode response"}`
	}
	if len(raw) > maxInvokeJSONBytes {
		return encodeInvokeFailure(invokeJSONSizeLimitMessage("response"))
	}
	return string(raw)
}

func invokeJSONSizeLimitMessage(kind string) string {
	return fmt.Sprintf(
		"invoke %s exceeds the %d MiB size limit",
		kind,
		maxInvokeJSONSizeMiB,
	)
}

func encodeInvokeFailure(message string) string {
	raw, err := json.Marshal(&invokeResponse{Err: message})
	if err != nil {
		return `{"success":false,"data":null,"error":"failed to encode response"}`
	}
	return string(raw)
}

func encodeInvokeNoDataResponse(err error) string {
	if err != nil {
		return encodeInvokeResponse(nil, err)
	}
	return encodeInvokeResponse(struct{}{}, nil)
}

func invokeGetFreePorts(payload json.RawMessage) string {
	request, err := decodePayload[GetFreePortsRequest](payload)
	if err != nil {
		return encodeInvokeResponse(nil, err)
	}
	ports, err := nodep.GetFreePorts(request.Count)
	if err != nil {
		return encodeInvokeResponse(nil, err)
	}
	return encodeInvokeResponse(&GetFreePortsResponse{Ports: ports}, nil)
}

func invokeConvertShareLinksToXrayJson(payload json.RawMessage) string {
	request, err := decodePayload[ConvertShareLinksToXrayJsonRequest](payload)
	if err != nil {
		return encodeInvokeResponse(nil, err)
	}
	xrayJson, err := share.ConvertShareLinksToXrayJson(request.Text)
	return encodeInvokeResponse(xrayJson, err)
}

func invokeConvertXrayJsonToShareLinks(payload json.RawMessage) string {
	request, err := decodePayload[ConvertXrayJsonToShareLinksRequest](payload)
	if err != nil {
		return encodeInvokeResponse(nil, err)
	}
	links, err := share.ConvertXrayJsonToShareLinks([]byte(request.XrayJson))
	if err != nil {
		return encodeInvokeResponse(nil, err)
	}
	return encodeInvokeResponse(&ConvertXrayJsonToShareLinksResponse{Links: links}, nil)
}

func invokeCountGeoData(payload json.RawMessage) string {
	request, err := decodePayload[CountGeoDataRequest](payload)
	if err != nil {
		return encodeInvokeNoDataResponse(err)
	}
	if request.DatDir == "" {
		return encodeInvokeNoDataResponse(errors.New("missing datDir"))
	}
	err = geo.CountGeoData(request.DatDir, request.Name, request.GeoType)
	return encodeInvokeNoDataResponse(err)
}

func invokeReadGeoFiles(payload json.RawMessage) string {
	request, err := decodePayload[ReadGeoFilesRequest](payload)
	if err != nil {
		return encodeInvokeResponse(nil, err)
	}
	domain, ip := geo.ReadGeoFiles([]byte(request.ConfigJSON))
	return encodeInvokeResponse(&ReadGeoFilesResponse{Domain: domain, IP: ip}, nil)
}

func invokePing(payload json.RawMessage) string {
	request, err := decodePayload[PingRequest](payload)
	if err != nil {
		return encodeInvokeResponse(nil, err)
	}
	delay, err := xray.Ping(request.ConfigPath, request.Timeout, request.URL, request.Proxy)
	if err != nil {
		if delay == nodep.PingDelayError || delay == nodep.PingDelayTimeout {
			return encodeInvokeResponse(&PingResponse{Delay: delay}, err)
		}
		return encodeInvokeResponse(nil, err)
	}
	return encodeInvokeResponse(&PingResponse{Delay: delay}, nil)
}

func invokeQueryStats(payload json.RawMessage) string {
	request, err := decodePayload[QueryStatsRequest](payload)
	if err != nil {
		return encodeInvokeResponse(nil, err)
	}
	stats, err := xray.QueryStats(request.Server)
	if err != nil {
		return encodeInvokeResponse(nil, err)
	}
	return encodeInvokeResponse(&QueryStatsResponse{Stats: stats}, nil)
}

func invokeQueryStatsByTag(payload json.RawMessage) string {
	request, err := decodePayload[QueryStatsByTagRequest](payload)
	if err != nil {
		return encodeInvokeResponse(nil, err)
	}
	stats, err := xray.QueryStatsByTag(request.Tag)
	if err != nil {
		return encodeInvokeResponse(nil, err)
	}
	return encodeInvokeResponse(&QueryStatsResponse{Stats: stats}, nil)
}

func invokeTestXray(payload json.RawMessage) string {
	request, err := decodePayload[RunXrayRequest](payload)
	if err != nil {
		return encodeInvokeNoDataResponse(err)
	}
	err = xray.TestXray(request.ConfigPath)
	return encodeInvokeNoDataResponse(err)
}

func invokeRunXray(payload json.RawMessage) string {
	request, err := decodePayload[RunXrayRequest](payload)
	if err != nil {
		return encodeInvokeNoDataResponse(err)
	}
	err = xray.RunXray(request.ConfigPath)
	return encodeInvokeNoDataResponse(err)
}

func invokeRunXrayFromJSON(payload json.RawMessage) string {
	request, err := decodePayload[RunXrayFromJSONRequest](payload)
	if err != nil {
		return encodeInvokeNoDataResponse(err)
	}
	err = xray.RunXrayFromJSON(request.ConfigJSON)
	return encodeInvokeNoDataResponse(err)
}

func invokeSetMemoryLimit(payload json.RawMessage) string {
	request, err := decodePayload[SetMemoryLimitRequest](payload)
	if err != nil {
		return encodeInvokeNoDataResponse(err)
	}
	memory.SetMemoryLimit(request.MemoryMB)
	return encodeInvokeNoDataResponse(nil)
}

func invokeReplaceInbound(payload json.RawMessage) string {
	request, err := decodePayload[InboundJSONRequest](payload)
	if err != nil {
		return encodeInvokeNoDataResponse(err)
	}
	return encodeInvokeNoDataResponse(xray.ReplaceInbound(request.InboundJSON))
}

func invokeReplaceOutbound(payload json.RawMessage) string {
	request, err := decodePayload[OutboundJSONRequest](payload)
	if err != nil {
		return encodeInvokeNoDataResponse(err)
	}
	return encodeInvokeNoDataResponse(xray.ReplaceOutbound(request.OutboundJSON))
}

func invokeAddInbound(payload json.RawMessage) string {
	request, err := decodePayload[InboundJSONRequest](payload)
	if err != nil {
		return encodeInvokeNoDataResponse(err)
	}
	return encodeInvokeNoDataResponse(xray.AddInbound(request.InboundJSON))
}

func invokeRemoveInbound(payload json.RawMessage) string {
	request, err := decodePayload[TagRequest](payload)
	if err != nil {
		return encodeInvokeNoDataResponse(err)
	}
	return encodeInvokeNoDataResponse(xray.RemoveInbound(request.Tag))
}

func invokeAddOutbound(payload json.RawMessage) string {
	request, err := decodePayload[OutboundJSONRequest](payload)
	if err != nil {
		return encodeInvokeNoDataResponse(err)
	}
	return encodeInvokeNoDataResponse(xray.AddOutbound(request.OutboundJSON))
}

func invokeRemoveOutbound(payload json.RawMessage) string {
	request, err := decodePayload[TagRequest](payload)
	if err != nil {
		return encodeInvokeNoDataResponse(err)
	}
	return encodeInvokeNoDataResponse(xray.RemoveOutbound(request.Tag))
}

func invokeAddRouteRule(payload json.RawMessage) string {
	request, err := decodePayload[AddRouteRuleRequest](payload)
	if err != nil {
		return encodeInvokeNoDataResponse(err)
	}
	return encodeInvokeNoDataResponse(xray.AddRouteRule(request.Rule, request.ShouldAppend))
}

func invokeRemoveRouteRule(payload json.RawMessage) string {
	request, err := decodePayload[RemoveRouteRuleRequest](payload)
	if err != nil {
		return encodeInvokeNoDataResponse(err)
	}
	return encodeInvokeNoDataResponse(xray.RemoveRouteRule(request.RuleTag))
}
