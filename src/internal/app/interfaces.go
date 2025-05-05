package app

import (
	api "github.com/osrg/gobgp/v3/api"
)

type controller interface {
	HandleUpdate(path *api.Path) error
	HandleWithdraw(path *api.Path) error
}
