package pkg

type Flags struct {
	FilePath     string
	FilePathsCap int
	Match        string
	Ignore       string
	DBPath       string
	PostAlways   string
	PostMin      string
	Log          string

	Min              int
	Every            uint64
	HealthCheckEvery uint64
	Proxy            string
	LogLevel         int
	MemLimit         int
	MSTeamsHook      string
	Anomaly          bool
	Test             bool
	Version          bool
}
