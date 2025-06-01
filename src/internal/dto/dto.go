package dto

import (
	"github.com/osrg/gobgp/v3/pkg/config/oc"
	"google.golang.org/protobuf/types/known/anypb"
)

type Evpn5Route struct {
	Rd           string
	RouteTargets []string
	Prefix       string
	Prefixlen    uint32
	Gateway      string
	Vni          uint32
	PathAttrs    []*anypb.Any
}

type VPNRoute struct {
	Rd           string
	RouteTargets []string
	Prefix       string
	Prefixlen    uint32
	PathAttrs    []*anypb.Any
}

type Vrf struct {
	Name               string
	Rd                 string
	ExportRouteTargets []string
	ImportRouteTargets []string
	Vni                uint32
}


type VrfDiff struct {
	Created []oc.VrfConfig
	Deleted []oc.VrfConfig
}
