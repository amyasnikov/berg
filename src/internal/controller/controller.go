package controller

import (
	"errors"
	"fmt"

	"github.com/amyasnikov/berg/internal/dto"
	"github.com/amyasnikov/berg/internal/utils"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	api "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/pkg/config/oc"
	"github.com/puzpuzpuz/xsync/v4"
)

// Handles updates and withdrawals of IPv4 routes
type VPNv4Controller struct {
	evpnInjector      evpnInjector
	rdVrfMap    *xsync.Map[string, dto.Vrf]
	redistributedEvpn *xsync.Map[vpnRoute, uuid.UUID]
	routeGen          *evpnRouteGen
}

func NewVPNv4Controller(injector evpnInjector, vrfCfg []oc.VrfConfig) *VPNv4Controller {
	vrfMap := make(map[string]oc.VrfConfig, len(vrfVfg))
	for _, vrf := range vrfVfg {
		vrfMap[vrf.Name] = vrf
	}
	neighborVrfMap := make(map[string]dto.Vrf, len(neighborCfg))
	for _, neighbor := range neighborCfg {
		vrf, ok := vrfMap[neighbor.Vrf]
		if !ok {
			continue
		}
		vrfDto := dto.Vrf{
			Name: vrf.Name,
			Rd: vrf.Rd,
			ImportRouteTargets: vrf.BothRtList,
			ExportRouteTargets: vrf.BothRtList,
			Vni: vrf.Id,
		}
		if len(vrf.ImportRtList) > 0 {
			vrfDto.ImportRouteTargets = vrf.ImportRtList
		}
		if len(vrf.ExportRtList) > 0 {
			vrfDto.ExportRouteTargets = vrf.ExportRtList
		}
		neighborVrfMap[neighbor.NeighborAddress] = vrfDto
	}
	return &IPv4Controller{
		evpnInjector: injector,
		neighborVrfMap: neighborVrfMap,
		redistributedEvpn: xsync.NewMap[ipv4Route, uuid.UUID](),
		routeGen: newEvpnRouteGen(),
	}
}


func (c *VPNv4Controller) HandleUpdate(path *api.Path) error {
	route, err := vpnFromApi(path.GetNlri())
	if err != nil {
		return err
	}
	vrf, ok := c.rdVrfMap.Load(route.Rd)
	if !ok {
		return nil
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

func (c *VPNv4Controller) HandleWithdraw(path *api.Path) error {
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
	rtVrfMap    map[string]string
	redistributedIPv4 *utils.MapSet[evpnRoute, ipv4RouteId]
	routeGen          *ipv4RouteGen
}

func NewEvpnController(injector ipv4Injector, vrfVfg []oc.VrfConfig) *EvpnController {
	rtmap := map[string]string{}
	for _, vrf := range vrfVfg {
		rtlist := vrf.BothRtList
		if len(vrf.ImportRtList) == 0 {
			rtlist = vrf.ImportRtList
		}
		for _, rt := range rtlist {
			rtmap[rt] = vrf.Name
		}
	}
	return &EvpnController{
		ipv4Injector: injector,
		rtVrfMap: rtmap,
		redistributedIPv4: utils.NewMapSet[evpnRoute, ipv4RouteId](),
		routeGen: newIPv4RouteGen(),
	}
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
			ipRoute := c.routeGen.genRoute(route, path.GetPattrs(), vrf)
			ipUuid, err := c.ipv4Injector.AddRoute(ipRoute)
			if err != nil {
				multierror.Append(merr, err)
				continue
			}
			rids = append(rids, ipv4RouteId{ipUuid, vrf})
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