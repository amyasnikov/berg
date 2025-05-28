package main

import (
	"context"
	"reflect"

	"github.com/amyasnikov/berg/internal/utils"
	api "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/pkg/config/oc"
	"github.com/osrg/gobgp/v3/pkg/server"
	"google.golang.org/protobuf/types/known/anypb"
)

func extractVrfConfig(vrfs []oc.Vrf) []oc.VrfConfig {
	vrfConfig := make([]oc.VrfConfig, 0, len(vrfs))
	for _, vrf := range vrfs {
		vrfConfig = append(vrfConfig, vrf.Config)
	}
	return vrfConfig
}

func getVrfDiff(old, new []oc.Vrf) ([]oc.VrfConfig, []oc.VrfConfig) {
	makeVrfMap := func(vrfs []oc.Vrf) map[uint32]oc.VrfConfig {
		result := make(map[uint32]oc.VrfConfig, len(vrfs))
		for _, vrf := range vrfs {
			result[vrf.Config.Id] = vrf.Config
		}
		return result
	}
	oldVrfConfig := makeVrfMap(old)
	newVrfConfig := makeVrfMap(new)
	deleted := []oc.VrfConfig{}
	created := []oc.VrfConfig{}
	for vrfId, oldVrf := range oldVrfConfig {
		newVrf, ok := newVrfConfig[vrfId]
		if !ok {
			deleted = append(deleted, oldVrf)
		} else if !reflect.DeepEqual(oldVrf, newVrf) {
			deleted = append(deleted, oldVrf)
			created = append(created, newVrf)
		}
		delete(newVrfConfig, vrfId)
	}
	for _, newVrf := range newVrfConfig {
		created = append(created, newVrf)
	}
	return created, deleted
}

func applyVrfChanges(bgpServer *server.BgpServer, created, deleted []oc.VrfConfig) error {
	for _, vrf := range deleted {
		req := api.DeleteVrfRequest{Name: vrf.Name}
		bgpServer.DeleteVrf(context.Background(), &req)
	}
	getRtList := func(rts []string) ([]*anypb.Any, error) {
		res := []*anypb.Any{}
		for _, rt := range rts {
			apirt, err := utils.RtToApi(rt)
			if err != nil {
				return nil, err
			}
			res = append(res, apirt)
		}
		return res, nil
	}
	for _, vrf := range created {
		rd, err := utils.RdToApi(vrf.Rd)
		if err != nil {
			return err
		}
		if len(vrf.ImportRtList) == 0 {
			vrf.ImportRtList = vrf.BothRtList
		}
		if len(vrf.ExportRtList) == 0 {
			vrf.ExportRtList = vrf.BothRtList
		}
		importRt, err := getRtList(vrf.ImportRtList)
		if err != nil {
			return err
		}
		exportRt, err := getRtList(vrf.ExportRtList)
		if err != nil {
			return err
		}
		req := &api.AddVrfRequest{Vrf: &api.Vrf{
			Id:       vrf.Id,
			Name:     vrf.Name,
			Rd:       rd,
			ImportRt: importRt,
			ExportRt: exportRt,
		}}
		bgpServer.AddVrf(context.Background(), req)
	}
	return nil
}
