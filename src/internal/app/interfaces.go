package app

import (
	"context"

	api "github.com/osrg/gobgp/v3/api"
)

type controller interface {
	HandleUpdate(path *api.Path) error
	HandleWithdraw(path *api.Path) error
}

type bgpServer interface {
	AddPath(context.Context, *api.AddPathRequest) (*api.AddPathResponse, error)
	DeletePath(context.Context, *api.DeletePathRequest) error
	WatchEvent(context.Context, *api.WatchEventRequest, func(*api.WatchEventResponse)) error
}
