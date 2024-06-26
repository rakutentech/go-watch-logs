package pkg

import (
	"log/slog"
	"os"
	"runtime"
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

func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	slog.Info("Memory Usage",
		"Alloc (MB)", bToMb(m.Alloc),
		"Sys", bToMb(m.Sys),
	)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}
