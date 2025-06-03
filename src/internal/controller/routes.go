package controller

import (
	"errors"
	"fmt"

	"github.com/amyasnikov/berg/internal/utils"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/google/uuid"
	api "github.com/osrg/gobgp/v3/api"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

var invalidEvpnType = errors.New("invalid EVPN type")

type ipv4RouteId struct {
	Uuid uuid.UUID
	Vrf  string
}

type ipv4Route struct {
	Prefix    string
	Prefixlen uint32
	Vrf       string
}

type evpnRoute struct {
	Rd          string
	Prefix      string
	Prefixlen   uint32
	Gateway     string
	Label       uint32
	EthernetTag uint32
	Esi         string
}

func (r evpnRoute) String() string {
	return fmt.Sprintf("5:%s:%s/%d Gw:%s Vni:%d", r.Rd, r.Prefix, r.Prefixlen, r.Gateway, r.Label)
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
	result.Rd, err = utils.RdToString(route.Rd)
	if err != nil {
		return evpnRoute{}, err
	}
	return result, nil
}

type EvpnRouteWithPattrs struct {
	Nlri    evpnRoute
	Pattrs  []*anypb.Any
	Targets mapset.Set[string]
}

func NewEvpnRouteWithPattrs(path *api.Path) (EvpnRouteWithPattrs, error) {
	route, err := evpnFromApi(path.GetNlri())
	if err != nil {
		return EvpnRouteWithPattrs{}, err
	}
	targets := extractRouteTargets(path.GetPattrs())
	return EvpnRouteWithPattrs{
		Nlri:    route,
		Pattrs:  path.GetPattrs(),
		Targets: mapset.NewSet(targets...),
	}, nil
}

func (r *EvpnRouteWithPattrs) HasAnyTarget(targets ...string) bool {
	return r.Targets.ContainsAny(targets...)
}

type vpnRoute struct {
	Rd        string
	Prefix    string
	Prefixlen uint32
	Label     uint32
}

func (r vpnRoute) String() string {
	return fmt.Sprintf("%s:%s/%d", r.Rd, r.Prefix, r.Prefixlen)
}

func vpnFromApi(apiRoute *anypb.Any) (vpnRoute, error) {
	var route api.LabeledVPNIPAddressPrefix
	err := anypb.UnmarshalTo(apiRoute, &route, proto.UnmarshalOptions{})
	if err != nil {
		return vpnRoute{}, invalidEvpnType
	}
	var result vpnRoute
	result.Label = route.Labels[0]
	result.Prefix = route.Prefix
	result.Prefixlen = route.PrefixLen
	result.Rd, err = utils.RdToString(route.Rd)
	if err != nil {
		return vpnRoute{}, err
	}
	return result, nil
}
