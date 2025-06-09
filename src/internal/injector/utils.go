package injector

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	api "github.com/osrg/gobgp/v3/api"
	"google.golang.org/protobuf/types/known/anypb"
)

func delRoute(server bgpServer, uuid uuid.UUID, family *api.Family) error {
	var nlri *anypb.Any
	rd, _ := anypb.New(&api.RouteDistinguisherTwoOctetASN{})
	if family.Safi == api.Family_SAFI_MPLS_VPN {
		nlri, _ = anypb.New(&api.LabeledVPNIPAddressPrefix{Rd: rd})
	} else if family.Safi == api.Family_SAFI_EVPN {
		nlri, _ = anypb.New(&api.EVPNIPPrefixRoute{Rd: rd, Esi: &api.EthernetSegmentIdentifier{Value: []byte{}}})
	} else {
		return fmt.Errorf("Unknown family %s", family.String())
	}
	nh, _ := anypb.New(&api.NextHopAttribute{NextHop: "0.0.0.0"})
	binUuid, _ := uuid.MarshalBinary()
	delReq := &api.DeletePathRequest{
		Uuid:   binUuid,
		Family: family,
		Path: &api.Path{
			Uuid:   binUuid,
			Family: family,
			Nlri:   nlri,
			Pattrs: []*anypb.Any{nh},
		},
	}

	if err := server.DeletePath(context.TODO(), delReq); err != nil {
		return err
	}
	return nil
}
