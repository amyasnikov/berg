package controller

import (
	"fmt"

	"github.com/osrg/gobgp/v3/pkg/log"

	"github.com/amyasnikov/gober/internal/dto"
	"github.com/google/uuid"
	api "github.com/osrg/gobgp/v3/api"
	"github.com/puzpuzpuz/xsync/v4"
)

type EventType int

const (
	ipv4Event EventType = iota
	evpnEvent
)

type Controller struct {
	ipv4Injector      ipv4Injector
	evpnInjector      evpnInjector
	neighborVrfMap    map[string]dto.Vrf
	redistributedIPv4 xsync.Map[evpnRoute, uuid.UUID]
	redistributedEvpn xsync.Map[ipv4Route, uuid.UUID]
	logger            log.Logger
}

func (c *Controller) HandleEvent(et EventType) func(*api.WatchEventResponse) {
	var updateFn, withdrawFn func(*api.Path)
	switch et {
	case ipv4Event:
		updateFn = c.handleIPv4Update
		withdrawFn = c.handleIPv4Withdraw
	case evpnEvent:
		updateFn = c.handleEVPNUpdate
		withdrawFn = c.handleEVPNWithdraw
	default:
		panic(fmt.Sprintf("invalid event type %d", et))
	}
	return func(event *api.WatchEventResponse) {
		for _, path := range event.GetTable().Paths {
			if path.GetNeighborIp() == "" {
				continue // locally originated route
			}
			if path.IsWithdraw {
				updateFn(path)
			} else {
				withdrawFn(path)
			}
		}
	}
}

func (c *Controller) handleIPv4Update(path *api.Path) {
	route, err := ipv4FromApi(path.GetNlri())
	if err != nil {
		c.logger.Error(err.Error(), log.Fields{})
	}
	vrf, ok := c.neighborVrfMap[path.GetNeighborIp()]
	if !ok {
		panic(fmt.Sprintf("Invalid configuration: neigbor %s has no associated VRF", path.GetNeighborIp()))
	}
	evpnRoute := genEvpnRoute(route, vrf, path.GetPattrs())
	evpnUuid, err := c.evpnInjector.AddType5Route(evpnRoute)
	if err != nil {
		c.logger.Error(err.Error(), log.Fields{})
	}
	if prevUuid, _ := c.redistributedEvpn.Load(route); prevUuid != uuid.Nil {
		c.evpnInjector.DelRoute(prevUuid) // implicit withdraw
	}
	c.redistributedEvpn.Store(route, evpnUuid)
}

func (c *Controller) handleIPv4Withdraw(path *api.Path) {
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

func (c *Controller) handleEVPNUpdate(path *api.Path) {

}

func (c *Controller) handleEVPNWithdraw(path *api.Path) {

}
