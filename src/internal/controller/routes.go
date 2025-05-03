package controller

import (
	"fmt"

	"google.golang.org/protobuf/types/known/anypb"
	api "github.com/osrg/gobgp/v3/api"
    "google.golang.org/protobuf/proto"

)


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
	
}