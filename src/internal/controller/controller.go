package controller

import (
	"fmt"

	"github.com/osrg/gobgp/v3/pkg/log"

	"github.com/amyasnikov/gober/internal/dto"
	"github.com/google/uuid"
	api "github.com/osrg/gobgp/v3/api"
	"github.com/puzpuzpuz/xsync/v4"
)


// Handles updates and withdrawals of IPv4 routes
type IPv4Controller struct {
	evpnInjector      evpnInjector
	neighborVrfMap    map[string]dto.Vrf
	redistributedEvpn xsync.Map[ipv4Route, uuid.UUID]
	routeGen          EvpnRouteGen
	logger            log.Logger
}


func (c *IPv4Controller) HandleUpdate(path *api.Path) {
	vrf, ok := c.neighborVrfMap[path.GetNeighborIp()]
	if !ok {
		panic(fmt.Sprintf("Invalid configuration: neigbor %s has no associated VRF", path.GetNeighborIp()))
	}
	route, err := ipv4FromApi(path.GetNlri())
	route.Vrf = vrf.Name
	if err != nil {
		c.logger.Error(err.Error(), log.Fields{})
	}
	evpnRoute := c.routeGen.GenRoute(route, vrf, path.GetPattrs())
	evpnUuid, err := c.evpnInjector.AddType5Route(evpnRoute)
	if err != nil {
		c.logger.Error(err.Error(), log.Fields{})
	}
	if prevUuid, _ := c.redistributedEvpn.Load(route); prevUuid != uuid.Nil {
		c.evpnInjector.DelRoute(prevUuid) // implicit withdraw
	}
	c.redistributedEvpn.Store(route, evpnUuid)
}

func (c *IPv4Controller) HandleWithdraw(path *api.Path) {
	route, err := ipv4FromApi(path.GetNlri())
	if err != nil {
		c.logger.Error(err.Error(), log.Fields{})
	}
	evpnUuid, _ := c.redistributedEvpn.Load(route)
	if evpnUuid != uuid.Nil {
		if err = c.evpnInjector.DelRoute(evpnUuid); err != nil {
			c.logger.Error(err.Error(), log.Fields{})
		}
		return
	}
	c.redistributedEvpn.Delete(route)
}


type EvpnController struct {
	ipv4Injector      evpnInjector
	neighborVrfMap    map[string]dto.Vrf
	redistributedIPv4 xsync.Map[evpnRoute, uuid.UUID]
	routeGen          ipv4RouteGen
	logger            log.Logger
}


func (c *EvpnController) handleUpdate(path *api.Path) {

}

func (c *EvpnController) handleWithdraw(path *api.Path) {

}
