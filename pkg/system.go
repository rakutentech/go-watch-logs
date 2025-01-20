package pkg

import (
	"log/slog"
	"os"
	"os/exec"
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
		slog.Error("Memory Limit Exceeded", "limit", f.MemLimit, "current", BToMb(m.Alloc))
		panic("Memory Limit Exceeded")
	}
}

func BToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func ExecShell(command string) (string, error) {
	if command == "" {
		return "", nil
	}
	cmd := exec.Command("sh", "-c", command)
	out, err := cmd.CombinedOutput()
	return string(out), err
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
