package gofernq

import (
	"github.com/fernq-org/gofernq/client"
	"github.com/fernq-org/gofernq/router"
)

type Client = client.Client
type Context = router.Context
type StateCode = router.StateCode
type Router struct {
	*router.Router
}

// Status codes.
const (
	StatusOK                  = router.StatusOK
	StatusCreated             = router.StatusCreated
	StatusAccepted            = router.StatusAccepted
	StatusNoContent           = router.StatusNoContent
	StatusBadRequest          = router.StatusBadRequest
	StatusUnauthorized        = router.StatusUnauthorized
	StatusForbidden           = router.StatusForbidden
	StatusNotFound            = router.StatusNotFound
	StatusConflict            = router.StatusConflict
	StatusPayloadTooLarge     = router.StatusPayloadTooLarge
	StatusTooManyRequests     = router.StatusTooManyRequests
	StatusInternalServerError = router.StatusInternalServerError
	StatusBadGateway          = router.StatusBadGateway
	StatusServiceUnavailable  = router.StatusServiceUnavailable
)

// NewClient creates a new client instance.
func NewClient(name string, more ...*Router) *Client {
	routers := make([]*router.Router, len(more))
	for i, r := range more {
		if r != nil {
			routers[i] = r.Router
		}
	}
	return client.NewClient(name, routers...)
}

// NewRouter creates a new router instance.
func NewRouter() *Router {
	return &Router{Router: router.NewRouter()}
}

// AddRoute adds a route to the router.
func (r *Router) AddRoute(path string, handler func(*Context)) error {
	return r.Router.AddRoute(path, func(c *router.Context) {
		handler(c)
	})
}
