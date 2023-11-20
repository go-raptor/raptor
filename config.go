package raptor

type Config struct {
	Address string
	Port    int
	Reload  bool
}

const (
	DefaultAddress = "127.0.0.1"
	DefaultPort    = 3000
	DefaultReload  = true
)

func config(userConfig ...Config) Config {
	config := Config{}

	if len(userConfig) > 0 {
		config = userConfig[0]
	}

	if config.Address == "" {
		config.Address = DefaultAddress
	}
	if config.Port == 0 {
		config.Port = DefaultPort
	}
	return config
}
