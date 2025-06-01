package controller

import (
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/google/uuid"
	"github.com/puzpuzpuz/xsync/v4"
)



type redistributedEvpnStorage struct {
	routeMap *xsync.Map[evpnRoute, uuid.UUID]
	rtMap *xsync.Map[string, mapset.Set[uuid.UUID]]
}

func (s *redistributedEvpnStorage) Store(route evpnRoute, targets []string, vpnUuid uuid.UUID) {
	s.routeMap.Store(route, vpnUuid)
	s.rtMap.Store()
}