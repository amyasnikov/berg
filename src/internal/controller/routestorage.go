package controller

import (
	"github.com/amyasnikov/berg/internal/utils"
	"github.com/google/uuid"
	"github.com/puzpuzpuz/xsync/v4"
)


type uuidRT struct {
	uuid uuid.UUID
	targets []string
}


type redistributedEvpnStorage struct {
	routeMap *xsync.Map[evpnRoute, uuidRT]
	rtMap *utils.MapSet[string, evpnRoute]
}

func newRedistributedEvpnStorage() *redistributedEvpnStorage {
	return &redistributedEvpnStorage{
		routeMap: xsync.NewMap[evpnRoute, uuidRT](),
		rtMap: utils.NewMapSet[string, evpnRoute](),
	}
}


func (s *redistributedEvpnStorage) Store(route evpnRoute, targets []string, vpnUuid uuid.UUID) {
	oldUuidRt, loaded := s.routeMap.LoadAndStore(route, uuidRT{uuid: vpnUuid, targets: targets})
	if loaded {
		for _, rt := range oldUuidRt.targets {
			s.rtMap.DeleteVal(rt, route)
		}
	}
	for _, rt := range targets {
		s.rtMap.Store(rt, route)
	}
}

func (s *redistributedEvpnStorage) Get(route evpnRoute) uuid.UUID {
	u, _ := s.routeMap.Load(route)
	return u.uuid
}

func (s *redistributedEvpnStorage) Delete(route evpnRoute, targets []string) {
	s.routeMap.Delete(route)
	for _, rt := range targets {
		s.rtMap.DeleteVal(rt, route)
	}
}

// get uuids and delete
func (s *redistributedEvpnStorage) PopByRT(targets []string) []uuid.UUID {
	uuids := []uuid.UUID{}
	for _, rt := range targets {
		routes, ok := s.rtMap.Load(rt)
		if !ok {
			continue
		}
		for route := range routes.Iter() {
			urt, loaded := s.routeMap.LoadAndDelete(route)
			if loaded {
				uuids = append(uuids, urt.uuid)
			}
		}
		s.rtMap.Delete(rt) 
	}
	return uuids
}
