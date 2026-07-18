// Package collision exists so core tests can build two distinct service
// types that share the bare type name CollisionService.
package collision

import "github.com/go-raptor/raptor/v4/core"

type CollisionService struct {
	core.Service
}
