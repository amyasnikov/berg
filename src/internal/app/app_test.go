package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/amyasnikov/berg/internal/dto"
	api "github.com/osrg/gobgp/v3/api"
	"github.com/osrg/gobgp/v3/pkg/config/oc"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/types/known/anypb"
)

// Mock for bgpServer interface
type mockBgpServer struct {
	mock.Mock
}

func (m *mockBgpServer) AddPath(ctx context.Context, req *api.AddPathRequest) (*api.AddPathResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(*api.AddPathResponse), args.Error(1)
}

func (m *mockBgpServer) DeletePath(ctx context.Context, req *api.DeletePathRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *mockBgpServer) WatchEvent(
	ctx context.Context,
	req *api.WatchEventRequest,
	fn func(*api.WatchEventResponse),
) error {
	args := m.Called(ctx, req, mock.AnythingOfType("func(*api.WatchEventResponse)"))
	return args.Error(0)
}

func (m *mockBgpServer) ListPath(ctx context.Context, r *api.ListPathRequest, fn func(*api.Destination)) error {
	args := m.Called(ctx, r, mock.AnythingOfType("func(*api.Destination)"))
	// Call the function to simulate the behavior, but pass empty destination
	go func() {
		fn(&api.Destination{})
	}()
	return args.Error(0)
}

// Mock for controller interface
type mockController struct {
	mock.Mock
}

func (m *mockController) HandleUpdate(path *api.Path) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *mockController) HandleWithdraw(path *api.Path) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *mockController) ReloadConfig(diff dto.VrfDiff) error {
	args := m.Called(diff)
	return args.Error(0)
}

// Helper function to create a test VPN path
func createTestVPNPath() *api.Path {
	rd, _ := anypb.New(&api.RouteDistinguisherTwoOctetASN{
		Admin:    65000,
		Assigned: 100,
	})

	nlri, _ := anypb.New(&api.LabeledVPNIPAddressPrefix{
		Labels:    []uint32{1000},
		Prefix:    "10.0.0.0",
		PrefixLen: 24,
		Rd:        rd,
	})

	return &api.Path{
		Family: &api.Family{
			Afi:  api.Family_AFI_IP,
			Safi: api.Family_SAFI_MPLS_VPN,
		},
		Nlri:       nlri,
		IsWithdraw: false,
		NeighborIp: "192.168.1.1",
	}
}

// Helper function to create a test EVPN path
func createTestEVPNPath() *api.Path {
	rd, _ := anypb.New(&api.RouteDistinguisherTwoOctetASN{
		Admin:    65000,
		Assigned: 100,
	})

	nlri, _ := anypb.New(&api.EVPNIPPrefixRoute{
		Rd:          rd,
		Esi:         &api.EthernetSegmentIdentifier{},
		EthernetTag: 0,
		IpPrefix:    "10.0.0.0",
		IpPrefixLen: 24,
		GwAddress:   "192.168.1.1",
		Label:       1000,
	})

	return &api.Path{
		Family: &api.Family{
			Afi:  api.Family_AFI_L2VPN,
			Safi: api.Family_SAFI_EVPN,
		},
		Nlri:       nlri,
		IsWithdraw: false,
		NeighborIp: "192.168.1.1",
	}
}

func TestApp_Sender(t *testing.T) {
	mockServer := &mockBgpServer{}
	logger := logrus.New()

	app := NewApp([]oc.VrfConfig{}, mockServer, 100, logger)

	// Create test response
	resp := &api.WatchEventResponse{}

	// Test that sender puts response in channel
	go app.sender(resp)

	// Read from channel with timeout
	select {
	case receivedResp := <-app.eventChan:
		assert.Equal(t, resp, receivedResp)
	case <-time.After(time.Millisecond * 100):
		t.Fatal("Response not received from channel")
	}
}

func TestApp_HandlePath(t *testing.T) {
	tests := []struct {
		name       string
		path       *api.Path
		isWithdraw bool
	}{
		{
			name:       "Handle update path",
			path:       createTestVPNPath(),
			isWithdraw: false,
		},
		{
			name:       "Handle withdraw path",
			path:       createTestVPNPath(),
			isWithdraw: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockServer := &mockBgpServer{}
			mockController := &mockController{}
			logger := logrus.New()

			app := NewApp([]oc.VrfConfig{}, mockServer, 100, logger)

			// Set withdraw status
			tt.path.IsWithdraw = tt.isWithdraw

			if tt.isWithdraw {
				mockController.On("HandleWithdraw", tt.path).Return(nil)
			} else {
				mockController.On("HandleUpdate", tt.path).Return(nil)
			}

			// Test the method
			app.handlePath(mockController, tt.path)

			mockController.AssertExpectations(t)
		})
	}
}

func TestApp_HandlePathWithError(t *testing.T) {
	mockServer := &mockBgpServer{}
	mockController := &mockController{}
	logger := logrus.New()

	app := NewApp([]oc.VrfConfig{}, mockServer, 100, logger)

	path := createTestVPNPath()
	expectedError := errors.New("handler error")

	// Mock controller to return error
	mockController.On("HandleUpdate", path).Return(expectedError)

	// Test that error is logged but doesn't cause panic
	app.handlePath(mockController, path)

	mockController.AssertExpectations(t)
}

func TestApp_ReloadConfig(t *testing.T) {
	mockServer := &mockBgpServer{}
	logger := logrus.New()

	app := NewApp([]oc.VrfConfig{}, mockServer, 100, logger)

	diff := dto.VrfDiff{
		Created: []oc.VrfConfig{{Name: "new-vrf"}},
		Deleted: []oc.VrfConfig{{Name: "old-vrf"}},
	}

	// Test that config reload message is sent
	go app.ReloadConfig(diff)

	// Read from control channel with timeout
	select {
	case msg := <-app.controlChan:
		assert.Equal(t, reloadConfigMsg, msg.Code)
		assert.Equal(t, &diff, msg.VrfDiff)
	case <-time.After(time.Millisecond * 100):
		t.Fatal("Config reload message not received")
	}
}

func TestApp_Serve_BasicFunctionality(t *testing.T) {
	mockServer := &mockBgpServer{}
	logger := logrus.New()

	// Mock WatchEvent to not return error - use simpler matching
	mockServer.On("WatchEvent", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	app := NewApp([]oc.VrfConfig{}, mockServer, 100, logger)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
	defer cancel()

	// Start serving in goroutine
	done := make(chan struct{})
	go func() {
		app.Serve(ctx)
		close(done)
	}()

	// Wait for completion with timeout
	select {
	case <-done:
		// Success
	case <-time.After(time.Second):
		t.Fatal("Serve did not complete after context cancellation")
	}

	mockServer.AssertExpectations(t)
}
