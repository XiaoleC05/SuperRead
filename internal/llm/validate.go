package llm

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

var allowedHosts = map[string]struct{}{
	"api.openai.com":    {},
	"api.anthropic.com": {},
	"openrouter.ai":     {},
	"api.deepseek.com":  {},
}

// ValidateAPIBase ensures apiBase targets an allowed public HTTPS LLM endpoint.
func ValidateAPIBase(apiBase string) error {
	apiBase = strings.TrimSpace(apiBase)
	if apiBase == "" {
		return fmt.Errorf("api base is required")
	}

	u, err := url.Parse(apiBase)
	if err != nil {
		return fmt.Errorf("invalid api base URL")
	}
	if u.Scheme != "https" {
		return fmt.Errorf("api base must use HTTPS")
	}

	host := strings.ToLower(u.Hostname())
	if host == "" {
		return fmt.Errorf("invalid api base host")
	}
	if host == "localhost" || strings.HasSuffix(host, ".local") {
		return fmt.Errorf("api base must not target local addresses")
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
			return fmt.Errorf("api base must not target private or loopback addresses")
		}
	}
	if _, ok := allowedHosts[host]; !ok {
		return fmt.Errorf("api base host not allowed")
	}
	return nil
}
