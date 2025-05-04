package controller

import (
	"errors"
	"fmt"

	"github.com/amyasnikov/gober/internal/dto"
	"github.com/amyasnikov/gober/internal/utils"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	api "github.com/osrg/gobgp/v3/api"
	"github.com/puzpuzpuz/xsync/v4"
)

// Handles updates and withdrawals of IPv4 routes
type IPv4Controller struct {
	evpnInjector      evpnInjector
	neighborVrfMap    map[string]dto.Vrf
	redistributedEvpn xsync.Map[ipv4Route, uuid.UUID]
	routeGen          EvpnRouteGen
}


func (c *IPv4Controller) HandleUpdate(path *api.Path) error {
	vrf, ok := c.neighborVrfMap[path.GetNeighborIp()]
	if !ok {
		panic(fmt.Sprintf("Invalid configuration: neigbor %s has no associated VRF", path.GetNeighborIp()))
	}
	route, err := ipv4FromApi(path.GetNlri())
	route.Vrf = vrf.Name
	if err != nil {
		return err
	}
	evpnRoute := c.routeGen.GenRoute(route, vrf, path.GetPattrs())
	evpnUuid, err := c.evpnInjector.AddType5Route(evpnRoute)
	if err != nil {
		return err
	}
	if prevUuid, _ := c.redistributedEvpn.Load(route); prevUuid != uuid.Nil {
		c.evpnInjector.DelRoute(prevUuid) // implicit withdraw
	}
	c.redistributedEvpn.Store(route, evpnUuid)
	return nil
}

func (c *IPv4Controller) HandleWithdraw(path *api.Path) error {
	route, err := ipv4FromApi(path.GetNlri())
	if err != nil {
		return err
	}
	evpnUuid, _ := c.redistributedEvpn.Load(route)
	if evpnUuid != uuid.Nil {
		c.redistributedEvpn.Delete(route)
		if err = c.evpnInjector.DelRoute(evpnUuid); err != nil {
			return err
		}
	}
	return nil
}


// Handles updates and withdrawals of EVPN routes
type EvpnController struct {
	ipv4Injector      ipv4Injector
	rtVrfMap    map[string]dto.Vrf
	redistributedIPv4 utils.MapSet[evpnRoute, ipv4RouteId]
	routeGen          ipv4RouteGen
}


func (c *EvpnController) HandleUpdate(path *api.Path) error {
	route, err := evpnFromApi(path.GetNlri())
	if errors.Is(err, invalidEvpnType) {
		return nil
	}
	if err != nil {
		return err
	}
	routeTargets := extractRouteTargets(path.GetPattrs())
	var merr error
	rids := []ipv4RouteId{}
	for _, rt := range routeTargets {
		if vrf, ok := c.rtVrfMap[rt]; ok {
			ipRoute := c.routeGen.genRoute(route, path.GetPattrs(), vrf.Name)
			ipUuid, err := c.ipv4Injector.AddRoute(ipRoute)
			if err != nil {
				multierror.Append(merr, err)
				continue
			}
			rids = append(rids, ipv4RouteId{ipUuid, vrf.Name})
		}
	}
	err = c.deleteRoute(route) // implicit withdraw
	if err != nil {
		multierror.Append(merr, err)
	}
	c.redistributedIPv4.StoreMany(route, rids)
	return merr
}

func (c *EvpnController) HandleWithdraw(path *api.Path) error {
	route, err := evpnFromApi(path.GetNlri())
	if errors.Is(err, invalidEvpnType) {
		return nil
	}
	if err != nil {
		return err
	}
	return c.deleteRoute(route)
}


func (c *EvpnController) deleteRoute(route evpnRoute) error {
	rids, ok := c.redistributedIPv4.Load(route)
	if !ok {
		return nil
	}
	var merr error
	for rid := range rids.Iter() {
		err := c.ipv4Injector.DelRoute(rid.Uuid, rid.Vrf)
		if err != nil {
			multierror.Append(merr, err)
		}
	}
	c.redistributedIPv4.Delete(route)
	return merr
}