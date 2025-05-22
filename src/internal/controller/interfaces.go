package controller

import (
	"github.com/amyasnikov/berg/internal/dto"
	"github.com/google/uuid"

)


type ipv4Injector interface {
	AddRoute(route dto.IPv4Route) (uuid.UUID, error)
	DelRoute(uuid uuid.UUID, vrf string) error
}


type evpnInjector interface {
	AddType5Route(route dto.Evpn5Route) (uuid.UUID, error)
	DelRoute(uuid uuid.UUID) error
}
