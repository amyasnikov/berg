package controller

import (
	"fmt"

	"google.golang.org/protobuf/types/known/anypb"
)


type ipv4Route struct {
	Prefix string
	Prefixlen uint32
	Vrf string
}


func ipv4FromApi(apiRoute *anypb.Any) (ipv4Route, error) {

}


func (r ipv4Route) String() string {
	return fmt.Sprintf("%s/%d", r.Prefix, r.Prefixlen)
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