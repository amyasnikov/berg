package controller

import (
	"errors"
	"sync"

	"github.com/amyasnikov/berg/internal/dto"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/google/uuid"
	multierror "github.com/hashicorp/go-multierror"
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
	return &VPNv4Controller{
		evpnInjector:      injector,
		rdVrfMap:          makeRdVrfMap(vrfCfg),
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

func (c *VPNv4Controller) ReloadConfig(diff dto.VrfDiff) error {
	deletedRd := make([]string, 0, len(diff.Deleted))
	for _, vrf := range diff.Deleted {
		c.rdVrfMap.Delete(vrf.Rd)
		deletedRd = append(deletedRd, vrf.Rd)
	}
	for _, vrf := range diff.Created {
		dtoVrf := dto.Vrf{
			Name:               vrf.Name,
			Vni:                vrf.Id,
			Rd:                 vrf.Rd,
			ImportRouteTargets: vrf.ImportRtList,
			ExportRouteTargets: vrf.ExportRtList,
		}
		c.rdVrfMap.Store(dtoVrf.Rd, dtoVrf)
	}
	return c.deleteStaleRoutes(deletedRd)
}

func (c *VPNv4Controller) deleteStaleRoutes(deletedRd []string) error {
	wg := sync.WaitGroup{}
	deletedSet := mapset.NewThreadUnsafeSet(deletedRd...)
	var merr error
	c.redistributedEvpn.Range(func(key vpnRoute, value uuid.UUID) bool {
		if deletedSet.Contains(key.Rd) {
			wg.Add(1)
			go func() {
				err := c.evpnInjector.DelRoute(value)
				c.redistributedEvpn.Delete(key)
				if err != nil {
					merr = multierror.Append(merr, err)
				}
				wg.Done()
			}()
		}
		return true
	})
	wg.Wait()
	return merr
}

// Handles updates and withdrawals of EVPN routes
type EvpnController struct {
	vpnInjector          vpnInjector
	existingRT           mapset.Set[string]
	redistributedStorage *redistributedEvpnStorage
	routeGen             *vpnRouteGen
	listEvpnRoutes       func() <-chan EvpnRouteWithPattrs
}

func NewEvpnController(
	injector vpnInjector, vrfCfg []oc.VrfConfig, listEvpnRoutes func() <-chan EvpnRouteWithPattrs,
) *EvpnController {
	existingRt := mapset.NewSet[string]()
	for _, vrf := range vrfCfg {
		existingRt.Append(vrf.ImportRtList...)
	}
	return &EvpnController{
		vpnInjector:          injector,
		existingRT:           existingRt,
		redistributedStorage: newRedistributedEvpnStorage(),
		routeGen:             newVpnRouteGen(),
		listEvpnRoutes:       listEvpnRoutes,
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
	if prevUuid := c.redistributedStorage.Get(route); prevUuid != uuid.Nil {
		c.vpnInjector.DelRoute(prevUuid) // implicit withdraw
	}
	c.redistributedStorage.Store(route, routeTargets, vpnUuid)
	return nil
}

func (c *EvpnController) HandleWithdraw(path *api.Path) error {
	route, err := evpnFromApi(path.GetNlri())
	if err != nil {
		return err
	}
	if vpnUuid := c.redistributedStorage.Get(route); vpnUuid != uuid.Nil {
		routeTargets := extractRouteTargets(path.GetPattrs())
		c.redistributedStorage.Delete(route, routeTargets)
		if err = c.vpnInjector.DelRoute(vpnUuid); err != nil {
			return err
		}
	}
	return nil
}

func (c *EvpnController) ReloadConfig(diff dto.VrfDiff) error {
	// modify c.existingRT
	deleteRT := []string{}
	createRT := []string{}
	for _, rt := range diff.Deleted {
		deleteRT = append(deleteRT, rt.ImportRtList...)
	}
	for _, rt := range diff.Created {
		createRT = append(createRT, rt.ImportRtList...)
	}
	c.existingRT.RemoveAll(deleteRT...)
	c.existingRT.Append(createRT...)

	// delete old VPN routes
	uuids := c.redistributedStorage.PopByRT(deleteRT)
	var merr error
	wg := sync.WaitGroup{}
	for _, rid := range uuids {
		rid := rid
		wg.Add(1)
		go func() {
			err := c.vpnInjector.DelRoute(rid)
			if err != nil {
				merr = multierror.Append(merr, err)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	// redistribute new vrfs
	ch := c.listEvpnRoutes()
	for route := range ch {
		if rid := c.redistributedStorage.Get(route.Nlri); rid != uuid.Nil {
			continue
		}
		if route.HasAnyTarget(createRT...) {
			vpnRoute := c.routeGen.GenRoute(route.Nlri, route.Pattrs)
			vpnRoute.RouteTargets = route.Targets.ToSlice()
			rid, err := c.vpnInjector.AddRoute(vpnRoute)
			merr = multierror.Append(merr, err)
			c.redistributedStorage.Store(route.Nlri, route.Targets.ToSlice(), rid)
		}
	}
	return merr
}
