package app

import (
	"context"
	api "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/pkg/config/oc"
	ctrl "github.com/amyasnikov/berg/internal/controller"
	"github.com/amyasnikov/berg/internal/injector"

)

type App struct {
	config         *oc.BgpConfigSet
	ipv4Controller controller
	evpnController controller
	eventChan      chan *api.WatchEventResponse
	bgpServer      bgpServer
}

func NewApp(config *oc.BgpConfigSet, bgpServer bgpServer, bufsize uint64) *App {
	ipv4Injector := injector.NewIPv4Injector(bgpServer)
	evpnInjector := injector.NewEvpnInjector(bgpServer)
	neighborConfig := make([]oc.NeighborConfig, 0, len(config.Neighbors))
	for _, neighbor := range config.Neighbors {
		neighborConfig = append(neighborConfig, neighbor.Config)
	}
	vrfConfig := make([]oc.VrfConfig, 0, len(config.Vrfs))
	for _, vrf := range config.Vrfs {
		vrfConfig = append(vrfConfig, vrf.Config)
	}
	ipv4Controller := ctrl.NewIPv4Controller(evpnInjector, neighborConfig, vrfConfig)
	evpnController := ctrl.NewEvpnController(ipv4Injector, vrfConfig)
	return &App{
		config: config,
		ipv4Controller: ipv4Controller,
		evpnController: evpnController,
		eventChan: make(chan *api.WatchEventResponse, bufsize),
		bgpServer: bgpServer,
	}
}

func (a *App) sender(resp *api.WatchEventResponse) {
	a.eventChan <- resp
}

func (a *App) receiver(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case resp, ok := <-a.eventChan:
			if !ok {
				return
			}
			for _, path := range resp.GetTable().GetPaths() {
				if path.NeighborIp == "" { // locally originated path
					continue
				}
				family := path.GetFamily()
				switch {
				case family.Afi == api.Family_AFI_IP && family.Safi == api.Family_SAFI_UNICAST:
					a.handlePath(a.ipv4Controller, path)
				case family.Afi == api.Family_AFI_L2VPN && family.Safi == api.Family_SAFI_EVPN:
					a.handlePath(a.evpnController, path)
				}
			}
		}
	}
}

func (a *App) handlePath(controller controller, path *api.Path) {
	if path.IsWithdraw {
		controller.HandleWithdraw(path)
	} else {
		controller.HandleUpdate(path)
	}
}

func (a *App) Serve(ctx context.Context) {
	watchReq := &api.WatchEventRequest{
		Table: &api.WatchEventRequest_Table{
			Filters: []*api.WatchEventRequest_Table_Filter{
				&api.WatchEventRequest_Table_Filter{
					Type: api.WatchEventRequest_Table_Filter_POST_POLICY,
					Init: true,
				},
			},
		},
	}
	a.bgpServer.WatchEvent(ctx, watchReq, a.sender)
	go a.receiver(ctx)
	<-ctx.Done()
	close(a.eventChan)
}
