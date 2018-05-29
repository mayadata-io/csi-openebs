package openebs

import (
	"github.com/openebs/csi-openebs/pkg/openebs/server"
	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/openebs/csi-openebs/pkg/openebs/driver"
)

type MockNonBlockingGRPCServer struct {
	server.NonBlockingGRPCServer
}

func (MockNonBlockingGRPCServer) Wait() {
}

func (MockNonBlockingGRPCServer) Start(endpoint string, ids csi.IdentityServer, cs csi.ControllerServer, ns csi.NodeServer) {
}

func MockNewNonBlockingGRPCServer() server.NonBlockingGRPCServer {
	return &MockNonBlockingGRPCServer{}
}

func TestNewIdentityServer(t *testing.T) {
	testCases := map[string]struct {
		driver *driver.CSIDriver
	}{
		"success": {
			driver: &driver.CSIDriver{Name: driverName, Version: "0.0.1", NodeID: "vm-1"},
		},
	}
	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			assert.Equal(t, v.driver, NewIdentityServer(v.driver).Driver)
		})
	}
}

func TestNewControllerServer(t *testing.T) {
	testCases := map[string]struct {
		driver *driver.CSIDriver
	}{
		"success": {
			driver: &driver.CSIDriver{Name: driverName, Version: "0.0.1", NodeID: "vm-1"},
		},
	}

	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			assert.Equal(t, v.driver, NewControllerServer(v.driver).Driver)
		})
	}
}

func TestNewNodeServer(t *testing.T) {
	testCases := map[string]struct {
		driver *driver.CSIDriver
	}{
		"success": {
			driver: &driver.CSIDriver{Name: driverName, Version: "0.0.1", NodeID: "vm-1"},
		},
	}
	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			assert.Equal(t, v.driver, NewNodeServer(v.driver).Driver)
		})
	}
}
func TestRun(t *testing.T) {
	nonBlockingGRPCServer = MockNewNonBlockingGRPCServer
	testCases := map[string]struct {
		plugin   *openEbs
		nodeId   string
		version  string
		endpoint string
	}{
		"success": {&openEbs{}, "vm-1", "0.0.1", "unix:///csi/csi.sock"},
	}

	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			v.plugin.Run(v.nodeId, v.endpoint)
			assert.Equal(t, v.endpoint, v.plugin.endpoint)
			assert.Equal(t, v.nodeId, v.plugin.driver.NodeID)
			assert.Equal(t, v.version, v.plugin.driver.Version)
		})
	}
}
