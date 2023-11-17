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
