package pkg

import (
	"log/slog"
	"os"
	"os/exec"
	"runtime"

	gmt "github.com/kevincobain2000/go-msteams/src"
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

func PrintMemUsage(f *Flags) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	slog.Info("Memory Usage",
		"Alloc (MB)", BToMb(m.Alloc),
		"Alloc (Bytes)", m.Alloc,
		"Sys (MB)", BToMb(m.Sys),
		"Sys (Bytes)", m.Sys,
		"NumGC", m.NumGC,
		"HeapAlloc (MB)", BToMb(m.HeapAlloc),
		"HeapSys (Bytes)", m.HeapSys,
	)
	if f.MemLimit > 0 && m.Alloc > uint64(f.MemLimit*1024*1024) {
		SendHealthCheck(f, &m)
		panic("Memory Limit Exceeded")
	}
}

func BToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func ExecShell(command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func SendHealthCheck(f *Flags, m *runtime.MemStats) {
	if f.MSTeamsHook == "" {
		return
	}
	details := GetPanicDetails(f, m)
	slog.Warn("Sending Panic Notify", "details", details)

	hostname, _ := os.Hostname()

	err := gmt.Send(hostname, details, f.MSTeamsHook, f.Proxy)
	if err != nil {
		slog.Error("Error sending to Teams", "error", err.Error())
	} else {
		slog.Info("Successfully sent to MS Teams")
	}
}
