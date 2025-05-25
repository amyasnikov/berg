package injector

import (
	"context"

	"github.com/amyasnikov/berg/internal/dto"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"
	api "github.com/osrg/gobgp/v3/api"
	"google.golang.org/protobuf/types/known/anypb"
)




type VPNv4Injector struct {
	s bgpServer
}


func NewVPNv4Injector(s bgpServer) *VPNv4Injector {
	return &VPNv4Injector{s: s}
}


func (c *VPNv4Injector) AddRoute(route dto.VPNv4Route) (uuid.UUID, error) {
    rd := mustParseRD(route.Rd)

	nlri := mustAny(&api.LabeledVPNIPAddressPrefix{
        Rd:           rd,
        Prefix:     route.Prefix,
        PrefixLen:  route.Prefixlen,
		Labels: []uint32{},
    })
    extcomms := make([]*anypb.Any, 0, len(route.RouteTargets))
    var merr error
    for _, rtString := range route.RouteTargets {
        rt, err := parseRT(rtString)
        multierror.Append(merr, err)
        extcomms = append(extcomms, rt)
    }
    if merr != nil {
        return uuid.Nil, merr
    }
    extcommAttr, _ := anypb.New(&api.ExtendedCommunitiesAttribute{
        Communities: extcomms,
    })
    pattrs := append(route.PathAttrs, extcommAttr)
    req := &api.AddPathRequest{
        Path: &api.Path{
            Family: &api.Family{
                Afi: api.Family_AFI_IP,
                Safi: api.Family_SAFI_MPLS_VPN,
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


func (c *VPNv4Injector) DelRoute(uuid uuid.UUID) error {
	family := &api.Family{
        Afi: api.Family_AFI_IP,
        Safi: api.Family_SAFI_MPLS_VPN,
	}
    return delRoute(c.s, uuid, family)
}
