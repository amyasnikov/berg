package utils

import (
	"testing"

	"github.com/osrg/gobgp/v3/pkg/config/oc"
	"github.com/stretchr/testify/assert"
)

func TestGetVrfDiff_EmptyInputs(t *testing.T) {
	old := []oc.VrfConfig{}
	new := []oc.VrfConfig{}

	diff := GetVrfDiff(old, new)

	assert.Empty(t, diff.Created)
	assert.Empty(t, diff.Deleted)
}

func TestGetVrfDiff_OnlyAdditions(t *testing.T) {
	old := []oc.VrfConfig{}
	new := []oc.VrfConfig{
		{
			Id:   1,
			Name: "vrf1",
			Rd:   "65000:1",
		},
		{
			Id:   2,
			Name: "vrf2",
			Rd:   "65000:2",
		},
	}

	diff := GetVrfDiff(old, new)

	assert.Len(t, diff.Created, 2)
	assert.Empty(t, diff.Deleted)
	assert.Contains(t, diff.Created, new[0])
	assert.Contains(t, diff.Created, new[1])
}

func TestGetVrfDiff_OnlyDeletions(t *testing.T) {
	old := []oc.VrfConfig{
		{
			Id:   1,
			Name: "vrf1",
			Rd:   "65000:1",
		},
		{
			Id:   2,
			Name: "vrf2",
			Rd:   "65000:2",
		},
	}
	new := []oc.VrfConfig{}

	diff := GetVrfDiff(old, new)

	assert.Empty(t, diff.Created)
	assert.Len(t, diff.Deleted, 2)
	assert.Contains(t, diff.Deleted, old[0])
	assert.Contains(t, diff.Deleted, old[1])
}

func TestGetVrfDiff_OnlyModifications(t *testing.T) {
	old := []oc.VrfConfig{
		{
			Id:   1,
			Name: "vrf1",
			Rd:   "65000:1",
		},
	}
	new := []oc.VrfConfig{
		{
			Id:   1,
			Name: "vrf1-modified",
			Rd:   "65000:1",
		},
	}

	diff := GetVrfDiff(old, new)

	assert.Len(t, diff.Created, 1)
	assert.Len(t, diff.Deleted, 1)
	assert.Equal(t, old[0], diff.Deleted[0])
	assert.Equal(t, new[0], diff.Created[0])
}

func TestGetVrfDiff_NoChanges(t *testing.T) {
	vrfConfig := []oc.VrfConfig{
		{
			Id:   1,
			Name: "vrf1",
			Rd:   "65000:1",
		},
		{
			Id:   2,
			Name: "vrf2",
			Rd:   "65000:2",
		},
	}
	old := vrfConfig
	new := make([]oc.VrfConfig, len(vrfConfig))
	copy(new, vrfConfig)

	diff := GetVrfDiff(old, new)

	assert.Empty(t, diff.Created)
	assert.Empty(t, diff.Deleted)
}

func TestGetVrfDiff_MixedScenario(t *testing.T) {
	old := []oc.VrfConfig{
		{
			Id:   1,
			Name: "vrf1",
			Rd:   "65000:1",
		},
		{
			Id:   2,
			Name: "vrf2",
			Rd:   "65000:2",
		},
		{
			Id:   3,
			Name: "vrf3",
			Rd:   "65000:3",
		},
		{
			Id:   4,
			Name: "vrf4",
			Rd:   "65000:4",
		},
	}
	new := []oc.VrfConfig{
		{
			Id:   1,
			Name: "vrf1",
			Rd:   "65000:1",
		}, // unchanged
		{
			Id:   2,
			Name: "vrf2-modified",
			Rd:   "65000:2",
		}, // modified
		// vrf3 and vrf4 deleted
		{
			Id:   5,
			Name: "vrf5",
			Rd:   "65000:5",
		}, // new
	}

	diff := GetVrfDiff(old, new)

	// Should have 2 created: modified vrf2 + new vrf5
	assert.Len(t, diff.Created, 2)
	// Should have 3 deleted: old vrf2 + deleted vrf3 + deleted vrf4
	assert.Len(t, diff.Deleted, 3)

	// Check created VRFs
	createdIds := make([]uint32, len(diff.Created))
	for i, vrf := range diff.Created {
		createdIds[i] = vrf.Id
	}
	assert.Contains(t, createdIds, uint32(2)) // modified vrf2
	assert.Contains(t, createdIds, uint32(5)) // new vrf5

	// Check deleted VRFs
	deletedIds := make([]uint32, len(diff.Deleted))
	for i, vrf := range diff.Deleted {
		deletedIds[i] = vrf.Id
	}
	assert.Contains(t, deletedIds, uint32(2)) // old vrf2
	assert.Contains(t, deletedIds, uint32(3)) // deleted vrf3
	assert.Contains(t, deletedIds, uint32(4)) // deleted vrf4
}
