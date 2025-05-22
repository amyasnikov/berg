package injector

import (
	"context"

	"github.com/amyasnikov/berg/internal/dto"
	"github.com/google/uuid"
	api "github.com/osrg/gobgp/v3/api"
	"google.golang.org/protobuf/types/known/anypb"
)

type IPv4Injector struct {
	s bgpServer
}

func NewIPv4Injector(s bgpServer) *IPv4Injector {
	return &IPv4Injector{s: s}
}

func (c *IPv4Injector) AddRoute(route dto.IPv4Route) (uuid.UUID, error) {
	nlri, err := anypb.New(&api.IPAddressPrefix{
		Prefix:    route.Prefix,
		PrefixLen: route.Prefixlen,
	})
	if err != nil {
		return uuid.Nil, err
	}
	addReq := &api.AddPathRequest{
		TableType: api.TableType_VRF,
		VrfId:     route.Vrf,
		Path: &api.Path{
			Family: &api.Family{
				Afi:  api.Family_AFI_IP,
				Safi: api.Family_SAFI_UNICAST,
			},
			Nlri:   nlri,
			Pattrs: route.PathAttrs,
		},
	}
	resp, err := c.s.AddPath(context.TODO(), addReq)
	if err != nil {
		return uuid.Nil, err
	}
	return uuid.FromBytes(resp.Uuid)
}

func (c *IPv4Injector) DelRoute(uuid uuid.UUID, vrf string) error {
	binUuid, _ := uuid.MarshalBinary()
	delReq := &api.DeletePathRequest{
		TableType: api.TableType_VRF,
		VrfId:     vrf,
		Family: &api.Family{
			Afi:  api.Family_AFI_IP,
			Safi: api.Family_SAFI_UNICAST,
		},
		Path: &api.Path{
			Uuid: binUuid,
		},
	}

	if err := c.s.DeletePath(context.TODO(), delReq); err != nil {
		return err
	}
	return nil
}
