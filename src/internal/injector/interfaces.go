package injector

import (
	"context"

	api "github.com/osrg/gobgp/v3/api"
)


type bgpServer interface {
	AddPath(context.Context, *api.AddPathRequest) (*api.AddPathResponse, error)
	DeletePath(context.Context, *api.DeletePathRequest) error
}
