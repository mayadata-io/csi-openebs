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

package driver

import (
	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"github.com/golang/glog"
)

type CSIDriver struct {
	Name    string
	NodeID  string
	Version string
	cap     []*csi.ControllerServiceCapability
	vc      []*csi.VolumeCapability_AccessMode
}

// Creates a NewCSIDriver object. Assumes vendor version is equal to driver version &
// does not support optional driver plugin info manifest field. Refer to CSI spec for more details.
func NewCSIDriver(name string, v string, nodeID string) *CSIDriver {
	if name == "" {
		glog.Errorf("Driver name missing")
		return nil
	}

	if nodeID == "" {
		glog.Errorf("NodeID missing")
		return nil
	}
	// TODO version format and validation
	if len(v) == 0 {
		glog.Errorf("Version argument missing")
		return nil
	}

	driver := CSIDriver{
		Name:    name,
		Version: v,
		NodeID:  nodeID,
	}

	return &driver
}

func (d *CSIDriver) AddControllerServiceCapabilities(cl []csi.ControllerServiceCapability_RPC_Type) {
	var csc []*csi.ControllerServiceCapability

	for _, c := range cl {
		glog.Infof("Enabling controller service capability: %v", c.String())

		cap := &csi.ControllerServiceCapability{
			Type: &csi.ControllerServiceCapability_Rpc{
				Rpc: &csi.ControllerServiceCapability_RPC{
					Type: c,
				},
			},
		}

		csc = append(csc, cap)
	}

	d.cap = csc

	return
}

func (d *CSIDriver) AddVolumeCapabilityAccessModes(vc []csi.VolumeCapability_AccessMode_Mode) []*csi.VolumeCapability_AccessMode {
	var vca []*csi.VolumeCapability_AccessMode
	for _, c := range vc {
		cap := &csi.VolumeCapability_AccessMode{Mode: c}
		glog.Infof("Enabling volume access mode: %v", c.String())
		vca = append(vca, cap)
	}
	d.vc = vca
	return vca
}

func (d *CSIDriver) GetControllerServiceCapability() []*csi.ControllerServiceCapability {
	return d.cap
}
