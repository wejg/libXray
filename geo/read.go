package geo

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/xtls/xray-core/infra/conf"
)

// ReadGeoFiles returns the geo data files referenced by an Xray config.
func ReadGeoFiles(configBytes []byte) ([]string, []string) {
	domain, ip := loadXrayConfig(configBytes)
	domainFiles := geoFileNames(domain, "geosite")
	ipFiles := geoFileNames(ip, "geoip")
	return domainFiles, ipFiles
}

func loadXrayConfig(configBytes []byte) ([]string, []string) {
	var config conf.Config
	if err := json.Unmarshal(configBytes, &config); err != nil {
		return []string{}, []string{}
	}

	domain, ip := filterRouting(&config)
	dnsDomain, dnsIP := filterDNS(&config)
	return append(domain, dnsDomain...), append(ip, dnsIP...)
}

func filterRouting(config *conf.Config) ([]string, []string) {
	if config.RouterConfig == nil {
		return []string{}, []string{}
	}

	domain := []string{}
	ip := []string{}
	type rawRule struct {
		Domain *conf.StringList `json:"domain"`
		IP     *conf.StringList `json:"ip"`
	}
	for _, rule := range config.RouterConfig.RuleList {
		var raw rawRule
		if err := json.Unmarshal(rule, &raw); err != nil {
			continue
		}
		if raw.Domain != nil {
			domain = append(domain, *raw.Domain...)
		}
		if raw.IP != nil {
			ip = append(ip, *raw.IP...)
		}
	}
	return domain, ip
}

func filterDNS(config *conf.Config) ([]string, []string) {
	if config.DNSConfig == nil {
		return []string{}, []string{}
	}

	domain := []string{}
	ip := []string{}
	for _, server := range config.DNSConfig.Servers {
		domain = append(domain, server.Domains...)
		ip = append(ip, server.ExpectIPs...)
	}
	return domain, ip
}

func geoFileNames(rules []string, defaultName string) []string {
	files := make(map[string]struct{})
	defaultPrefix := defaultName + ":"
	defaultFile := defaultName + ".dat"
	for _, rule := range rules {
		if strings.HasPrefix(rule, defaultPrefix) {
			files[defaultFile] = struct{}{}
			continue
		}
		if strings.HasPrefix(rule, "ext:") {
			parts := strings.SplitN(rule, ":", 3)
			if len(parts) == 3 && parts[1] != "" {
				files[parts[1]] = struct{}{}
			}
		}
	}

	result := make([]string, 0, len(files))
	for file := range files {
		result = append(result, file)
	}
	sort.Strings(result)
	return result
}
