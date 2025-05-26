package controller

import (
	"errors"

	"github.com/amyasnikov/berg/internal/dto"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/google/uuid"
	api "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/pkg/config/oc"
	"github.com/puzpuzpuz/xsync/v4"
)

// Handles updates and withdrawals of IPv4 routes
type VPNv4Controller struct {
	evpnInjector      evpnInjector
	rdVrfMap          *xsync.Map[string, dto.Vrf]
	redistributedEvpn *xsync.Map[vpnRoute, uuid.UUID]
	routeGen          *evpnRouteGen
}

func NewVPNv4Controller(injector evpnInjector, vrfCfg []oc.VrfConfig) *VPNv4Controller {
	rdVrfMap := xsync.NewMap[string, dto.Vrf]()
	for _, vrf := range vrfCfg {
		vrfDto := dto.Vrf{
			Name:               vrf.Name,
			Rd:                 vrf.Rd,
			ImportRouteTargets: vrf.BothRtList,
			ExportRouteTargets: vrf.BothRtList,
			Vni:                vrf.Id,
		}
		if len(vrf.ImportRtList) > 0 {
			vrfDto.ImportRouteTargets = vrf.ImportRtList
		}
		if len(vrf.ExportRtList) > 0 {
			vrfDto.ExportRouteTargets = vrf.ExportRtList
		}
		rdVrfMap.Store(vrfDto.Rd, vrfDto)
	}
	return &VPNv4Controller{
		evpnInjector:      injector,
		rdVrfMap:          rdVrfMap,
		redistributedEvpn: xsync.NewMap[vpnRoute, uuid.UUID](),
		routeGen:          newEvpnRouteGen(),
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
	evpnRoute, err := c.routeGen.GenRoute(route, vrf, path.GetPattrs())
	if err != nil {
		return err
	}
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
	route, err := vpnFromApi(path.GetNlri())
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
	vpnInjector      vpnInjector
	existingRT       mapset.Set[string]
	redistributedVpn *xsync.Map[evpnRoute, uuid.UUID]
	routeGen         *vpnRouteGen
}

func NewEvpnController(injector vpnInjector, vrfCfg []oc.VrfConfig) *EvpnController {
	existingRt := mapset.NewSet[string]()
	for _, vrf := range vrfCfg {
		existingRt.Append(vrf.ImportRtList...)
	}
	return &EvpnController{
		vpnInjector:      injector,
		existingRT:       existingRt,
		redistributedVpn: xsync.NewMap[evpnRoute, uuid.UUID](),
		routeGen:         newVpnRouteGen(),
	}
}

func (c *EvpnController) HandleUpdate(path *api.Path) error {
	route, err := evpnFromApi(path.GetNlri())
	if errors.Is(err, invalidEvpnType) { // TODO: conditionally support Type-2
		return nil
	}
	if err != nil {
		return err
	}
	routeTargets := extractRouteTargets(path.GetPattrs())
	if !c.existingRT.ContainsAny(routeTargets...) {
		return nil
	}
	vpnRoute := c.routeGen.GenRoute(route, path.GetPattrs())
	vpnRoute.RouteTargets = routeTargets
	vpnUuid, err := c.vpnInjector.AddRoute(vpnRoute)
	if err != nil {
		return err
	}
	if prevUuid, _ := c.redistributedVpn.Load(route); prevUuid != uuid.Nil {
		c.vpnInjector.DelRoute(prevUuid) // implicit withdraw
	}
	c.redistributedVpn.Store(route, vpnUuid)
	return nil
}

func (c *EvpnController) HandleWithdraw(path *api.Path) error {
	route, err := evpnFromApi(path.GetNlri())
	if err != nil {
		return err
	}
	vpnUuid, _ := c.redistributedVpn.Load(route)
	if vpnUuid != uuid.Nil {
		c.redistributedVpn.Delete(route)
		if err = c.vpnInjector.DelRoute(vpnUuid); err != nil {
			return err
		}
	}
	return nil
}
