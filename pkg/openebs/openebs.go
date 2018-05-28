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

const (
	driverName = "csi-openebs"
)

var (
	version = "0.0.1"
)

func GetOpenEbsDriver() *openEbs {
	return &openEbs{}
}

func (oe *openEbs) Run(nodeID, endpoint string) {
	// Initialize with default driver
	oe.driver = driver.NewCSIDriver(driverName, version, nodeID)
	if oe.driver == nil {
		glog.Fatalln("Failed to initialize CSI Driver.")
	}

	// Add capabilities
	oe.driver.AddControllerServiceCapabilities([]csi.ControllerServiceCapability_RPC_Type{csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME})
	oe.driver.AddVolumeCapabilityAccessModes([]csi.VolumeCapability_AccessMode_Mode{csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER})

	oe.ids = NewIdentityServer(oe.driver)
	oe.ns = NewNodeServer(oe.driver)
	oe.cs = NewControllerServer(oe.driver)

	// Create GRPC server and starts it
	s := server.NewNonBlockingGRPCServer()
	s.Start(endpoint, oe.ids, oe.cs, oe.ns)
	s.Wait()
}

func NewIdentityServer(d *driver.CSIDriver) *server.IdentityServer {
	return &server.IdentityServer{
		Driver: d,
	}
}

func NewControllerServer(d *driver.CSIDriver) *server.ControllerServer {
	return &server.ControllerServer{
		Driver: d,
	}
}

func NewNodeServer(d *driver.CSIDriver) *server.NodeServer {
	return &server.NodeServer{
		Driver: d,
	}
}
