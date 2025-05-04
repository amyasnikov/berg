package controller

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	api "github.com/osrg/gobgp/v3/api"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)


var invalidEvpnType = errors.New("invalid EVPN type")
var invalidRD = errors.New("invalid RD")

type ipv4RouteId struct {
	Uuid uuid.UUID
	Vrf string
}

type ipv4Route struct {
	Prefix string
	Prefixlen uint32
	Vrf string
}


func ipv4FromApi(apiRoute *anypb.Any) (ipv4Route, error) {
	var ip api.IPAddressPrefix
	err := anypb.UnmarshalTo(apiRoute, &ip, proto.UnmarshalOptions{})
	if err == nil {
		return ipv4Route{
			Prefix: ip.Prefix,
			Prefixlen: ip.PrefixLen,
		}, nil
	}
	return ipv4Route{}, err
}


func (r ipv4Route) String() string {
	return fmt.Sprintf("%s/%d vrf %s", r.Prefix, r.Prefixlen, r.Vrf)
}




type evpnRoute struct {
	Rd string
	Prefix string
	Prefixlen uint32
	Gateway string
	Label uint32
	EthernetTag uint32	
	Esi string
}


func (r evpnRoute) String() string {
	return fmt.Sprintf("5:") // TODO
}


func evpnFromApi(apiRoute *anypb.Any) (evpnRoute, error) {
	var route api.EVPNIPPrefixRoute
	err := anypb.UnmarshalTo(apiRoute, &route, proto.UnmarshalOptions{})
	if err != nil {
		return evpnRoute{}, invalidEvpnType
	}
	var result evpnRoute
	result.Esi = route.Esi.String()
	result.EthernetTag = route.EthernetTag
	result.Gateway = route.GwAddress
	result.Label = route.Label
	result.Prefix = route.IpPrefix
	result.Prefixlen = route.IpPrefixLen
	rd1 := api.RouteDistinguisherTwoOctetASN{}
	rd2 := api.RouteDistinguisherFourOctetASN{}
	rd3 := api.RouteDistinguisherIPAddress{}
	if err = route.Rd.UnmarshalTo(&rd1); err == nil {
		result.Rd = rd1.String()
	} else if err = route.Rd.UnmarshalTo(&rd2); err == nil {
		result.Rd = rd2.String()
	} else if err = route.Rd.UnmarshalTo(&rd3); err == nil {
		result.Rd = rd3.String()
	} else {
		return evpnRoute{}, invalidRD
	}
	return result, nil
}
