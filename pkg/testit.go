package pkg

import (
	"log/slog"
	"regexp"
)

func TestIt(filepath string, match string) {
	fps, err := FilesByPattern(filepath, false)
	if err != nil {
		slog.Error("Error finding files", "error", err.Error())
	}
	slog.Info("Files found", "count", len(fps))
	for _, filePath := range fps {
		slog.Info("Found file", "filePath", filePath)
	}
	str := ReadFromPipeInput()
	if str == "" {
		slog.Error("No input found, see --help for usage")
		return
	}
	str = str[:len(str)-1] // strip new line
	re, err := regexp.Compile(match)
	if err != nil {
		slog.Error("Error compiling regex", "error", err.Error())
		return
	}
	if re.Match([]byte(str)) {
		slog.Info("Matched", "Match Regex", match, "input", str, "Match Found", re.FindString(str))
	} else {
		slog.Warn("Not matched", "Match", match, "str", str)
	}
}
