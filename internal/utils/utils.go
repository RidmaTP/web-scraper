package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"

	"github.com/RidmaTP/web-analyzer/internal/models"
)

// utility functions are included here

func ContainsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func JsonToText(output models.Output) (*string, error) {
	jsonBytes, err := json.Marshal(output)
	if err != nil {
		return nil, errors.New("err when marshelling : " + err.Error())
	}
	str := string(jsonBytes)
	return &str, nil
}

func ErrStreamObj(err models.ErrorOut) *string {
	errString := fmt.Sprintf(`{"error" : "%s", "status_code" : "%d"}`, err.Error, err.StatusCode)
	return &errString
}

func IsExternalLink(link, baseUrl string) bool {
	u, err := url.Parse(link)
	if err != nil {
		return false
	}
	bu, err := url.Parse(baseUrl)
	if err != nil {
		return false
	}
	if u.Host == "" {
		return false
	}
	return !strings.Contains(u.Host, bu.Host)
}

func AddInternalHost(link, baseUrl string) string {
	u, _ := url.Parse(link)
	bu, _ := url.Parse(baseUrl)

	if u.Host == "" {
		return bu.Scheme + "://" + bu.Host + link
	}
	return link
}

func UrlValidationCheck(input *string) *models.ErrorOut {

	if len(strings.Split(*input, "://")) == 1 {
		*input = "https://" + *input
	}
	errOut := models.ErrorOut{StatusCode: http.StatusBadRequest, Error: "invalid url"}
	parsedVal, err := url.ParseRequestURI(*input)
	if err != nil {
		return &errOut
	}
	if parsedVal.Scheme != "http" && parsedVal.Scheme != "https" {
		errOut.Error = "url scheme not found"
		return &errOut
	}
	hostStr := ""
	if strings.Contains(parsedVal.Host, "www.") {
		wwwRemoved := strings.Split(parsedVal.Host, "www.")[1]
		hostStr = wwwRemoved
	} else {
		hostStr = parsedVal.Host
	}
	if !strings.Contains(hostStr, ".") {
		errOut.Error = "url domain not found"
		return &errOut
	}

	if parsedVal.Host == "" {
		errOut.Error = "url host not found"
		return &errOut
	}

	return nil
}

func ValidateRequestURL(ctx context.Context, input *string) *models.ErrorOut {
	if errObj := UrlValidationCheck(input); errObj != nil {
		return errObj
	}
	return ValidateInitialURL(ctx, *input)
}

// ValidateInitialURL applies the SSRF policy to the URL that will be opened by Rod.
func ValidateInitialURL(ctx context.Context, input string) *models.ErrorOut {
	parsed, err := url.Parse(input)
	if err != nil || parsed.Hostname() == "" {
		return &models.ErrorOut{StatusCode: http.StatusBadRequest, Error: "invalid url"}
	}

	if ip, err := netip.ParseAddr(parsed.Hostname()); err == nil {
		if isBlockedIP(ip) {
			return &models.ErrorOut{StatusCode: http.StatusBadRequest, Error: "url destination is not allowed"}
		}
		return nil
	}

	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", parsed.Hostname())
	if err != nil {
		return &models.ErrorOut{StatusCode: http.StatusBadRequest, Error: "url host could not be resolved"}
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return &models.ErrorOut{StatusCode: http.StatusBadRequest, Error: "url destination is not allowed"}
		}
	}
	return nil
}

func isBlockedIP(ip netip.Addr) bool {
	if !ip.IsValid() {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() ||
		ip.IsInterfaceLocalMulticast() {
		return true
	}

	blockedPrefixes := []string{
		"0.0.0.0/8",
		"100.64.0.0/10",
		"192.0.0.0/24",
		"192.0.2.0/24",
		"198.18.0.0/15",
		"198.51.100.0/24",
		"203.0.113.0/24",
		"240.0.0.0/4",
		"::/128",
		"64:ff9b::/96",
		"100::/64",
		"2001:db8::/32",
		"2001::/32",
		"fc00::/7",
	}
	for _, prefix := range blockedPrefixes {
		network, err := netip.ParsePrefix(prefix)
		if err == nil && network.Contains(ip) {
			return true
		}
	}
	return false
}
