package core

import "net/http"

type Core struct {
	Utils *Utils
}

func NewCore(utils *Utils) *Core {
	return &Core{
		Utils: utils,
	}
}

func (c *Core) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Raptor is running"}`))
}
