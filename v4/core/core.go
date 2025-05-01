package core

import (
	"sync"
)

type Core struct {
	Resources   *Resources
	Handlers    map[string]map[string]*Handler
	Services    map[string]ServiceInitializer
	ContextPool *sync.Pool
}

func NewCore(resources *Resources) *Core {
	binder := &DefaultBinder{}

	return &Core{
		Resources: resources,
		Handlers:  make(map[string]map[string]*Handler),
		Services:  make(map[string]ServiceInitializer),
		ContextPool: &sync.Pool{
			New: func() interface{} {
				return NewContext(nil, nil, binder)
			},
		},
	}
}
