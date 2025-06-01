package utils

import (
	"reflect"

	"github.com/amyasnikov/berg/internal/dto"
	"github.com/osrg/gobgp/v3/pkg/config/oc"
)


func GetVrfDiff(old, new []oc.VrfConfig) dto.VrfDiff {
	makeVrfMap := func(vrfs []oc.VrfConfig) map[uint32]oc.VrfConfig {
		result := make(map[uint32]oc.VrfConfig, len(vrfs))
		for _, vrf := range vrfs {
			result[vrf.Id] = vrf
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
	return dto.VrfDiff{
		Created: created,
		Deleted: deleted,
	}
}