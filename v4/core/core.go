package core

import (
	"sync"

	"github.com/go-raptor/raptor/v4/components"
)

type Core struct {
	Resources   *components.Resources
	Handlers    map[string]map[string]*Handler
	ContextPool *sync.Pool
}

func NewCore(resources *components.Resources) *Core {
	return &Core{
		Resources: resources,
		Handlers:  make(map[string]map[string]*Handler),
		ContextPool: &sync.Pool{
			New: func() interface{} {
				return NewContext(nil, nil)
			},
		},
	}
}
