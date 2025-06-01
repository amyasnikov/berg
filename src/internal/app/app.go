package app

import (
	"context"
	"fmt"

	ctrl "github.com/amyasnikov/berg/internal/controller"
	"github.com/amyasnikov/berg/internal/dto"
	"github.com/amyasnikov/berg/internal/injector"
	api "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/pkg/config/oc"
	"github.com/sirupsen/logrus"
)

type App struct {
	vpnController  controller
	evpnController controller
	eventChan      chan *api.WatchEventResponse
	controlChan chan message
	bgpServer      bgpServer
	logger         *logrus.Logger
}

func NewApp(vrfConfig []oc.VrfConfig, bgpServer bgpServer, bufsize uint64, logger *logrus.Logger) *App {
	vpnInjector := injector.NewVPNv4Injector(bgpServer)
	evpnInjector := injector.NewEvpnInjector(bgpServer)
	vpnController := ctrl.NewVPNv4Controller(evpnInjector, vrfConfig)
	evpnController := ctrl.NewEvpnController(vpnInjector, vrfConfig)
	return &App{
		vpnController:  vpnController,
		evpnController: evpnController,
		eventChan:      make(chan *api.WatchEventResponse, bufsize),
		controlChan: make(chan message, 1),
		bgpServer:      bgpServer,
		logger:         logger,
	}
}

func (a *App) sender(resp *api.WatchEventResponse) {
	a.eventChan <- resp
}

func (a *App) receiver() {
	for {
		select {
		case msg := <- a.controlChan:
			switch msg.Code {
			case stopAppMsg:
				return
			case reloadConfigMsg:
				a.evpnController.ReloadConfig(*msg.VrfDiff)
				a.vpnController.ReloadConfig(*msg.VrfDiff)
			default:
				a.logger.Error(fmt.Sprintf("Invalid message from controlChan: ", msg))
			}
		case resp, ok := <-a.eventChan:
			if !ok {
				return
			}
			for _, path := range resp.GetTable().GetPaths() {
				if path.NeighborIp == "" || path.NeighborIp == "<nil>" { // locally originated path
					continue
				}
				family := path.GetFamily()
				switch {
				case family.Afi == api.Family_AFI_IP && family.Safi == api.Family_SAFI_MPLS_VPN:
					a.handlePath(a.vpnController, path)
				case family.Afi == api.Family_AFI_L2VPN && family.Safi == api.Family_SAFI_EVPN:
					a.handlePath(a.evpnController, path)
				}
			}
		}
	}
}

func (a *App) handlePath(controller controller, path *api.Path) {
	if a.logger.IsLevelEnabled(logrus.DebugLevel) {
		a.logger.WithFields(logrus.Fields{"path": path.String()}).Debug("received path")
	}
	var handler func(*api.Path) error
	if path.IsWithdraw {
		handler = controller.HandleWithdraw
	} else {
		handler = controller.HandleUpdate
	}
	if err := handler(path); err != nil {
		a.logger.Error(err.Error())
	}
}

func (a *App) Serve(ctx context.Context) {
	watchReq := &api.WatchEventRequest{
		Table: &api.WatchEventRequest_Table{
			Filters: []*api.WatchEventRequest_Table_Filter{
				{
					Type: api.WatchEventRequest_Table_Filter_BEST,
					Init: true,
				},
			},
		},
	}
	a.bgpServer.WatchEvent(ctx, watchReq, a.sender)
	go func ()  {
		<- ctx.Done()
		close(a.controlChan)
	}()
	go a.receiver()
	<-ctx.Done()
	close(a.eventChan)
}

func (a *App) ReloadConfig(diff dto.VrfDiff) {
	a.controlChan <- message{
		Code: reloadConfigMsg,
		VrfDiff: &diff,
	}
}
