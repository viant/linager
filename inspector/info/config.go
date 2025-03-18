package info

type Config struct {
	IncludeUnexported bool
	SkipTests         bool
	RecursivePackages bool
	SkipAsset         bool //
}

func DefaultConfig() *Config {
	return &Config{
		IncludeUnexported: true,
		SkipTests:         false,
		RecursivePackages: true,
	}
}
