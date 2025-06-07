package injector

import (
	"context"

	api "github.com/osrg/gobgp/v3/api"
	"github.com/stretchr/testify/mock"
)

// Mock implementation of bgpServer for tests
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
