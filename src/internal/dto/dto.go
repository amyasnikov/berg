package dto

import (
	"google.golang.org/protobuf/types/known/anypb"
)


type IPv4Route struct {
	Prefix string
	Prefixlen uint32
	Nexthop string
	PathAttrs []*anypb.Any
	Vrf string
}


type Evpn5Route struct {
	Rd string
	RouteTargets []string
	Prefix string
	Prefixlen uint32
	Gateway string
	Vni uint32
	PathAttrs []*anypb.Any
}


type VPNv4Route struct {
	Rd string
	RouteTargets []string
	Prefix string
	Prefixlen uint32
	PathAttrs []*anypb.Any
}

type Vrf struct {
	Name string
	Rd string
	ExportRouteTargets []string
	ImportRouteTargets []string
	Vni uint32
}