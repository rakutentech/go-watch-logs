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
	slog.Debug("Memory Usage",
		"Alloc (MB)", BToMb(m.Alloc),
		"Alloc (Bytes)", m.Alloc,
		"Sys (MB)", BToMb(m.Sys),
		"Sys (Bytes)", m.Sys,
		"NumGC", m.NumGC,
		"HeapAlloc (MB)", BToMb(m.HeapAlloc),
		"HeapSys (Bytes)", m.HeapSys,
	)
	if f.MemLimit > 0 && m.Alloc > uint64(f.MemLimit)*1024*1024 {
		sendPanicCheck(f, &m)
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

func sendPanicCheck(f *Flags, m *runtime.MemStats) {
	details := GetPanicDetails(f, m)
	var logDetails []interface{} // nolint: prealloc
	for _, detail := range details {
		logDetails = append(logDetails, detail.Label, detail.Message)
	}
	slog.Warn("Sending Panic Check", logDetails...)

	if f.MSTeamsHook == "" {
		slog.Warn("MS Teams hook not set")
		return
	}

	hostname, _ := os.Hostname()

	err := gmt.Send(hostname, details, f.MSTeamsHook, f.Proxy)
	if err != nil {
		slog.Error("Error sending to Teams", "error", err.Error())
	} else {
		slog.Info("Successfully sent to MS Teams")
	}
}

func ReadFromPipeInput() string {
	fi, _ := os.Stdin.Stat()
	if (fi.Mode() & os.ModeCharDevice) == 0 {
		buf := make([]byte, 0, 4096)
		tmp := make([]byte, 256)
		for {
			n, err := os.Stdin.Read(tmp)
			if n == 0 {
				break
			}
			if err != nil {
				slog.Error("Error reading from pipe", "error", err.Error())
				break
			}
			buf = append(buf, tmp[:n]...)
		}
		return string(buf)
	}
	return ""
}
