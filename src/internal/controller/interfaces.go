package controller

import (
	"github.com/amyasnikov/berg/internal/dto"
	"github.com/google/uuid"
)

type vpnInjector interface {
	AddRoute(route dto.VPNRoute) (uuid.UUID, error)
	DelRoute(uuid uuid.UUID) error
}

type evpnInjector interface {
	AddType5Route(route dto.Evpn5Route) (uuid.UUID, error)
	DelRoute(uuid uuid.UUID) error
}
