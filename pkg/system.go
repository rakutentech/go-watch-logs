package pkg

import (
	"os"
)

func SystemProxy() string {
	proxyVars := []string{"https_proxy", "http_proxy", "HTTPS_PROXY", "HTTP_PROXY"}

	for _, proxyVar := range proxyVars {
		if os.Getenv(proxyVar) != "" {
			return os.Getenv(proxyVar)
		}
	}
	return ""
}
