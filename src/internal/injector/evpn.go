package injector

import (
	"context"

	"github.com/amyasnikov/gober/internal/dto"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	api "github.com/osrg/gobgp/v3/api"
	"google.golang.org/protobuf/types/known/anypb"
)




type EvpnInjector struct {
	s bgpServer
}


func NewEvpnInjector(s bgpServer) *EvpnInjector {
	return &EvpnInjector{s: s}
}


func (c *EvpnInjector) AddType5Route(route dto.Evpn5Route) (uuid.UUID, error) {
    rd := mustParseRD(route.Rd)

	nlri := mustAny(&api.EVPNIPPrefixRoute{
        Rd:           rd,
        Esi:          &api.EthernetSegmentIdentifier{},
        EthernetTag:  0,
        IpPrefix:     route.Prefix,
        IpPrefixLen:  route.Prefixlen,
        GwAddress:    route.Gateway,
        Label:        uint32(route.Vni),
    })
    extcomms := make([]*anypb.Any, 0, len(route.RouteTargets) + 1)
    var merr error
    for _, rtString := range route.RouteTargets {
        rt, err := parseRT(rtString)
        multierror.Append(merr, err)
        extcomms = append(extcomms, rt)
    }
    if merr != nil {
        return uuid.Nil, merr
    }
    encap := mustAny(&api.EncapExtended{TunnelType: 1}) // VXLAN encap
    extcommAttr, _ := anypb.New(&api.ExtendedCommunitiesAttribute{
        Communities: append(extcomms, encap),
    })
    pattrs := append(route.PathAttrs, extcommAttr)
    req := &api.AddPathRequest{
        Path: &api.Path{
            Family: &api.Family{
                Afi: api.Family_AFI_L2VPN,
                Safi: api.Family_SAFI_EVPN,
            },
            Nlri: nlri,
            Pattrs: pattrs,
        },
    }
    resp, err := c.s.AddPath(context.TODO(), req)
    if  err != nil {
        return uuid.Nil, err
    }
    return uuid.FromBytes(resp.Uuid)
}


func (c *EvpnInjector) DelRoute(uuid []byte) error {
	delReq := &api.DeletePathRequest{
		Family: &api.Family{
            Afi: api.Family_AFI_L2VPN,
            Safi: api.Family_SAFI_EVPN,
		},
		Path: &api.Path{
			Uuid: uuid,
		},
	}

	if err := c.s.DeletePath(context.TODO(), delReq); err != nil {
		return err
	}
	return nil
}