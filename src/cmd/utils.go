package main

import (
	"context"

	"github.com/amyasnikov/berg/internal/dto"
	"github.com/amyasnikov/berg/internal/utils"
	api "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/pkg/config/oc"
	"google.golang.org/protobuf/types/known/anypb"
)

type VrfManager interface {
	DeleteVrf(context.Context, *api.DeleteVrfRequest) error
	AddVrf(context.Context, *api.AddVrfRequest) error
}

func extractVrfConfig(vrfs []oc.Vrf) []oc.VrfConfig {
	vrfConfig := make([]oc.VrfConfig, 0, len(vrfs))
	for _, vrf := range vrfs {
		vrfConfig = append(vrfConfig, vrf.Config)
	}
	return vrfConfig
}

func getVrfDiff(old, new []oc.Vrf) dto.VrfDiff {
	getCfg := func(vrfs []oc.Vrf) []oc.VrfConfig {
		configs := make([]oc.VrfConfig, 0, len(vrfs))
		for _, vrf := range vrfs {
			configs = append(configs, vrf.Config)
		}
		return configs
	}
	oldcfg := getCfg(old)
	newcfg := getCfg(new)
	return utils.GetVrfDiff(oldcfg, newcfg)
}

func applyVrfChanges(bgpServer VrfManager, created, deleted []oc.VrfConfig) error {
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
