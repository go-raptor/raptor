package router

import (
	"net/http"

	"github.com/go-raptor/raptor/v4/core"
)

type routeHandler struct {
	core       *core.Core
	controller string
	action     string
	path       string
}

func (rh *routeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rh.core.Handler(w, r, rh.controller, rh.action, rh.path)
}
