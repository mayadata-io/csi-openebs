/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package openebs

import (
	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"github.com/golang/glog"
	"github.com/openebs/csi-openebs/pkg/openebs/driver"
	"github.com/openebs/csi-openebs/pkg/openebs/server"
)

type openEbs struct {
	driver   *driver.CSIDriver
	endpoint string

	ids *server.IdentityServer
	ns  *server.NodeServer
	cs  *server.ControllerServer
}

// function to create NonBlockingGRPCServer implementing struct object
type NewNonBlockingGRPCServer func() server.NonBlockingGRPCServer

const (
	driverName = "csi-openebs"
)

var (
	version               string
	nonBlockingGRPCServer NewNonBlockingGRPCServer
)

func init() {
	version = "0.0.1"
	nonBlockingGRPCServer = server.NewNonBlockingGRPCServer
}

func GetOpenEbsDriver() *openEbs {
	return &openEbs{}
}

// Run initializes openEbs driver and creates a grpc server
func (oe *openEbs) Run(nodeID, endpoint string) {
	glog.Infof("Creating driver object")
	oe.endpoint = endpoint
	// Initialize with default driver
	oe.driver = driver.NewCSIDriver(driverName, version, nodeID)
	if oe.driver == nil {
		glog.Fatalln("Failed to initialize CSI Driver.")
	}

	// Add capabilities
	glog.V(3).Infof("Adding controller service capabilities")
	oe.driver.AddControllerServiceCapabilities([]csi.ControllerServiceCapability_RPC_Type{csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME})
	glog.V(3).Infof("Adding volume capability access modes")
	oe.driver.AddVolumeCapabilityAccessModes([]csi.VolumeCapability_AccessMode_Mode{csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER})

	glog.V(4).Infof("Initializing Identity, Node and Controller servers")
	oe.ids = NewIdentityServer(oe.driver)
	oe.ns = NewNodeServer(oe.driver)
	oe.cs = NewControllerServer(oe.driver)

	glog.V(4).Infof("IdentityServer=%v, NodeServer=%v,ControllerServer=%v", oe.ids, oe.ns, oe.cs)

	// Create GRPC server and starts it
	glog.Infof("Starting GRPC server")
	s := nonBlockingGRPCServer()
	s.Start(oe.endpoint, oe.ids, oe.cs, oe.ns)
	s.Wait()
}

// NewIdentityServer creates and returns and IdentityServer pointer
func NewIdentityServer(d *driver.CSIDriver) *server.IdentityServer {
	return &server.IdentityServer{
		Driver: d,
	}
}

// NewControllerServer creates and returns and ControllerServer pointer
func NewControllerServer(d *driver.CSIDriver) *server.ControllerServer {
	return &server.ControllerServer{
		Driver: d,
	}
}

// NewNodeServer creates and returns and NodeServer pointer
func NewNodeServer(d *driver.CSIDriver) *server.NodeServer {
	return &server.NodeServer{
		Driver: d,
	}
}
