package injector

import (
	"context"

	"github.com/amyasnikov/berg/internal/dto"
	"github.com/amyasnikov/berg/internal/utils"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	api "github.com/osrg/gobgp/v3/api"
	"google.golang.org/protobuf/types/known/anypb"
)

type VPNInjector struct {
	s   bgpServer
	afi api.Family_Afi
}

func NewVPNv4Injector(s bgpServer) *VPNInjector {
	return &VPNInjector{s: s, afi: api.Family_AFI_IP}
}

func (c *VPNInjector) AddRoute(route dto.VPNRoute) (uuid.UUID, error) {
	rd, err := utils.RdToApi(route.Rd)
	if err != nil {
		return uuid.Nil, err
	}

	nlri, _ := anypb.New(&api.LabeledVPNIPAddressPrefix{
		Rd:        rd,
		Prefix:    route.Prefix,
		PrefixLen: route.Prefixlen,
		Labels:    []uint32{0},
	})
	extcomms := make([]*anypb.Any, 0, len(route.RouteTargets))
	var merr error
	for _, rtString := range route.RouteTargets {
		rt, err := utils.RtToApi(rtString)
		if err != nil {
			merr = multierror.Append(merr, err)
		}
		extcomms = append(extcomms, rt)
	}
	if merr != nil {
		return uuid.Nil, merr
	}
	extcommAttr, _ := anypb.New(&api.ExtendedCommunitiesAttribute{
		Communities: extcomms,
	})
	nh , _ := anypb.New(&api.NextHopAttribute{NextHop: "0.0.0.0"})
	pattrs := append(route.PathAttrs, extcommAttr, nh)
	req := &api.AddPathRequest{
		Path: &api.Path{
			Family: &api.Family{
				Afi:  c.afi,
				Safi: api.Family_SAFI_MPLS_VPN,
			},
			Nlri:   nlri,
			Pattrs: pattrs,
		},
	}
	resp, err := c.s.AddPath(context.TODO(), req)
	if err != nil {
		return uuid.Nil, err
	}
	return uuid.FromBytes(resp.Uuid)
}

func (c *VPNInjector) DelRoute(uuid uuid.UUID) error {
	family := &api.Family{
		Afi:  c.afi,
		Safi: api.Family_SAFI_MPLS_VPN,
	}
	return delRoute(c.s, uuid, family)
}
