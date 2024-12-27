package pkg

type Flags struct {
	FilePath     string
	FilePathsCap int
	Match        string
	Ignore       string
	DBPath       string
	PostAlways   string
	PostCommand  string
	LogFile      string

	Min               int
	Every             uint64
	HealthCheckEvery  uint64
	Proxy             string
	LogLevel          int
	MemLimit          int
	MSTeamsHook       string
	Anomaly           bool
	AnomalyWindowDays int
	NotifyOnlyRecent  bool
	Test              bool
	Version           bool
}
