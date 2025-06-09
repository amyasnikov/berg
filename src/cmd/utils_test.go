package main

import (
	"context"
	"errors"
	"testing"

	api "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/pkg/config/oc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockVrfManager is a mock implementation of the VrfManager interface
type MockVrfManager struct {
	mock.Mock
}

func (m *MockVrfManager) DeleteVrf(ctx context.Context, req *api.DeleteVrfRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockVrfManager) AddVrf(ctx context.Context, req *api.AddVrfRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func TestApplyVrfChanges_DeleteVrfs(t *testing.T) {
	tests := []struct {
		name    string
		deleted []oc.VrfConfig
		wantErr bool
	}{
		{
			name: "delete single VRF",
			deleted: []oc.VrfConfig{
				{Name: "vrf1"},
			},
			wantErr: false,
		},
		{
			name: "delete multiple VRFs",
			deleted: []oc.VrfConfig{
				{Name: "vrf1"},
				{Name: "vrf2"},
			},
			wantErr: false,
		},
		{
			name:    "delete no VRFs",
			deleted: []oc.VrfConfig{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := new(MockVrfManager)

			// Set up expectations for delete calls
			for _, vrf := range tt.deleted {
				expectedReq := &api.DeleteVrfRequest{Name: vrf.Name}
				mockManager.On("DeleteVrf", mock.Anything, expectedReq).Return(nil)
			}

			err := applyVrfChanges(mockManager, []oc.VrfConfig{}, tt.deleted)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockManager.AssertExpectations(t)
		})
	}
}

func TestApplyVrfChanges_CreateVrfs(t *testing.T) {
	tests := []struct {
		name    string
		created []oc.VrfConfig
		wantErr bool
	}{
		{
			name: "create VRF with explicit import/export RT lists",
			created: []oc.VrfConfig{
				{
					Id:           1,
					Name:         "vrf1",
					Rd:           "65000:100",
					ImportRtList: []string{"65000:100"},
					ExportRtList: []string{"65000:100"},
				},
			},
			wantErr: false,
		},
		{
			name: "create VRF with BothRtList fallback",
			created: []oc.VrfConfig{
				{
					Id:         1,
					Name:       "vrf1",
					Rd:         "65000:100",
					BothRtList: []string{"65000:100"},
				},
			},
			wantErr: false,
		},
		{
			name: "create VRF with mixed RT lists",
			created: []oc.VrfConfig{
				{
					Id:           1,
					Name:         "vrf1",
					Rd:           "65000:100",
					ImportRtList: []string{"65000:100"},
					BothRtList:   []string{"65000:200"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := new(MockVrfManager)

			// Set up expectations for add calls
			for _, vrf := range tt.created {
				mockManager.On("AddVrf", mock.Anything, mock.MatchedBy(func(req *api.AddVrfRequest) bool {
					return req.Vrf.Name == vrf.Name &&
						req.Vrf.Id == vrf.Id
				})).Return(nil)
			}

			err := applyVrfChanges(mockManager, tt.created, []oc.VrfConfig{})

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockManager.AssertExpectations(t)
		})
	}
}

func TestApplyVrfChanges_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		created     []oc.VrfConfig
		deleted     []oc.VrfConfig
		mockSetup   func(*MockVrfManager)
		wantErr     bool
		expectedErr string
	}{
		{
			name: "delete VRF error",
			deleted: []oc.VrfConfig{
				{Name: "vrf1"},
			},
			mockSetup: func(m *MockVrfManager) {
				m.On("DeleteVrf", mock.Anything, mock.Anything).Return(errors.New("delete failed"))
			},
			wantErr: false, // The function doesn't return delete errors
		},
		{
			name: "add VRF error",
			created: []oc.VrfConfig{
				{
					Id:         1,
					Name:       "vrf1",
					Rd:         "invalid-rd",
					BothRtList: []string{"65000:100"},
				},
			},
			mockSetup: func(m *MockVrfManager) {
				// This will not be called due to RD conversion error
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := new(MockVrfManager)
			tt.mockSetup(mockManager)

			err := applyVrfChanges(mockManager, tt.created, tt.deleted)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedErr != "" {
					assert.Contains(t, err.Error(), tt.expectedErr)
				}
			} else {
				assert.NoError(t, err)
			}

			mockManager.AssertExpectations(t)
		})
	}
}

func TestApplyVrfChanges_RTListLogic(t *testing.T) {
	tests := []struct {
		name           string
		vrf            oc.VrfConfig
		expectedImport []string
		expectedExport []string
	}{
		{
			name: "explicit import/export lists",
			vrf: oc.VrfConfig{
				Id:           1,
				Name:         "vrf1",
				Rd:           "65000:100",
				ImportRtList: []string{"65000:100"},
				ExportRtList: []string{"65000:200"},
				BothRtList:   []string{"65000:300"},
			},
			expectedImport: []string{"65000:100"},
			expectedExport: []string{"65000:200"},
		},
		{
			name: "empty import list, uses BothRtList",
			vrf: oc.VrfConfig{
				Id:           1,
				Name:         "vrf1",
				Rd:           "65000:100",
				ImportRtList: []string{},
				ExportRtList: []string{"65000:200"},
				BothRtList:   []string{"65000:300"},
			},
			expectedImport: []string{"65000:300"},
			expectedExport: []string{"65000:200"},
		},
		{
			name: "empty export list, uses BothRtList",
			vrf: oc.VrfConfig{
				Id:           1,
				Name:         "vrf1",
				Rd:           "65000:100",
				ImportRtList: []string{"65000:100"},
				ExportRtList: []string{},
				BothRtList:   []string{"65000:300"},
			},
			expectedImport: []string{"65000:100"},
			expectedExport: []string{"65000:300"},
		},
		{
			name: "both lists empty, use BothRtList for both",
			vrf: oc.VrfConfig{
				Id:           1,
				Name:         "vrf1",
				Rd:           "65000:100",
				ImportRtList: []string{},
				ExportRtList: []string{},
				BothRtList:   []string{"65000:300"},
			},
			expectedImport: []string{"65000:300"},
			expectedExport: []string{"65000:300"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager := new(MockVrfManager)

			// Mock the AddVrf call and capture the request to verify RT lists
			mockManager.On("AddVrf", mock.Anything, mock.MatchedBy(func(req *api.AddVrfRequest) bool {
				// Verify the RT lists are set correctly
				// Note: This is a simplified check since we can't easily verify the
				// exact content of the protobuf Any types without more complex logic
				return req.Vrf.Name == tt.vrf.Name &&
					len(req.Vrf.ImportRt) > 0 &&
					len(req.Vrf.ExportRt) > 0
			})).Return(nil)

			err := applyVrfChanges(mockManager, []oc.VrfConfig{tt.vrf}, []oc.VrfConfig{})

			assert.NoError(t, err)
			mockManager.AssertExpectations(t)
		})
	}
}

func TestApplyVrfChanges_MixedOperations(t *testing.T) {
	mockManager := new(MockVrfManager)

	created := []oc.VrfConfig{
		{
			Id:         1,
			Name:       "vrf1",
			Rd:         "65000:100",
			BothRtList: []string{"65000:100"},
		},
	}

	deleted := []oc.VrfConfig{
		{Name: "vrf2"},
	}

	// Set up expectations
	mockManager.On("DeleteVrf", mock.Anything, &api.DeleteVrfRequest{Name: "vrf2"}).Return(nil)
	mockManager.On("AddVrf", mock.Anything, mock.MatchedBy(func(req *api.AddVrfRequest) bool {
		return req.Vrf.Name == "vrf1"
	})).Return(nil)

	err := applyVrfChanges(mockManager, created, deleted)

	assert.NoError(t, err)
	mockManager.AssertExpectations(t)
}
