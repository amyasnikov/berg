package controller

import (
	"github.com/amyasnikov/berg/internal/dto"
	"github.com/google/uuid"

)


type vpnv4Injector interface {
	AddRoute(route dto.VPNv4Route) (uuid.UUID, error)
	DelRoute(uuid uuid.UUID, vrf string) error
}


type evpnInjector interface {
	AddType5Route(route dto.Evpn5Route) (uuid.UUID, error)
	DelRoute(uuid uuid.UUID) error
}
