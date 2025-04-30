package router

import (
	"net/http"

	"github.com/go-raptor/raptor/v4/core"
)

type routeHandler struct {
	core       *core.Core
	controller string
	action     string
}

func (rh *routeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rh.core.Handle(w, r, rh.controller, rh.action)
}
