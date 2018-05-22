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

package server

import (
	"golang.org/x/net/context"
	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"github.com/princerachit/csi-openebs/pkg/openebs/mayaproxy"
	mayav1 "github.com/kubernetes-incubator/external-storage/openebs/types/v1"
	"github.com/golang/glog"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/codes"
	"fmt"
	"encoding/json"
	"github.com/princerachit/csi-openebs/pkg/openebs/driver"
)

type ControllerServer struct {
	csi.ControllerServer
	Driver *driver.CSIDriver
}

func checkArguments(req *csi.CreateVolumeRequest) error {
	if len(req.GetName()) == 0 {
		return status.Error(codes.InvalidArgument, "Name missing in request")
	}
	if req.GetVolumeCapabilities() == nil {
		return status.Error(codes.InvalidArgument, "Volume Capabilities missing in request")
	}
	if req.Parameters["storage-class-name"] == "" {
		return status.Error(codes.InvalidArgument, "Missing storage-class-name in request")
	}
	return nil
}

// getVolumeAttributes iterates over volume's annotation and returns a map of attributes
func getVolumeAttributes(volume *mayav1.Volume) (map[string]string) {
	var iqn, targetPortal, portals string
	var portalList []string
	for key, value := range volume.Metadata.Annotations.(map[string]interface{}) {
		switch key {
		case "vsm.openebs.io/iqn":
			iqn = value.(string)
		case "vsm.openebs.io/targetportals":
			targetPortal = value.(string)
		case "openebs.io/jiva-target-portal":
			portalList = []string{value.(string)}
		}
	}

	marshaledPortalList, _ := json.Marshal(portalList)

	portals = string(marshaledPortalList)
	// values hardcoded below. Do they need fix?
	attributes := map[string]string{"iqn": iqn, "targetPortal": targetPortal, "lun": "0", "portals": portals, "iscsiInterface": "default"}
	return attributes
}

// createVolumeSpec returns a volume spec created from the req object
func createVolumeSpec(req *csi.CreateVolumeRequest) (mayav1.VolumeSpec) {
	volumeSpec := mayav1.VolumeSpec{}

	// Convert Bytes to Gigabyte
	volSize := int64(req.GetCapacityRange().GetRequiredBytes() / 1e9)
	volumeSpec.Metadata.Labels.Storage = fmt.Sprintf("%dG", volSize)
	volumeSpec.Metadata.Labels.StorageClass = req.Parameters["storage-class-name"]
	volumeSpec.Metadata.Name = req.Name
	volumeSpec.Metadata.Labels.Namespace = "default"

	return volumeSpec
}

func (cs *ControllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	glog.Infof("Received request: %v", req)

	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME); err != nil {
		glog.V(3).Infof("invalid create volume req: %v", req)
		return nil, err
	}

	// Check arguments
	err := checkArguments(req)
	if err != nil {
		return nil, err
	}

	mayaConfig := mayaproxy.GetNewMayaConfig()
	err = mayaConfig.SetupMayaConfig(mayaproxy.K8sClient{})
	if err != nil {
		glog.Errorf("Error setting up MayaConfig")
		return nil, status.Error(codes.Unavailable, fmt.Sprint(err))
	}

	var volume *mayav1.Volume

	// If volume retrieval fails then create the volume
	volume, err = mayaConfig.ListVolume(req.GetName())
	if err != nil {
		volumeSpec := createVolumeSpec(req)

		glog.Info("Attempting to create volume")
		err = mayaConfig.CreateVolume(volumeSpec)

		if err != nil {
			return nil, status.Error(codes.Unavailable, fmt.Sprint(err))
		}
	}

	volume, err = mayaConfig.ListVolume(req.Name)
	if err != nil {
		return nil, status.Error(codes.DeadlineExceeded, fmt.Sprintf("Unable to contact amapi server: %v", err))
	}

	glog.V(2).Infof("[DEBUG] Volume details %s", volume)
	glog.V(2).Infof("[DEBUG] Volume metadata %v", volume.Metadata)

	attributes := getVolumeAttributes(volume)

	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			Id:            volume.Metadata.Name,
			CapacityBytes: req.GetCapacityRange().GetRequiredBytes(),
			Attributes:    attributes,
		},
	}, nil
}

func (cs *ControllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	glog.Infof("Received request: %v", req)
	if err := cs.Driver.ValidateControllerServiceRequest(csi.ControllerServiceCapability_RPC_CREATE_DELETE_VOLUME); err != nil {
		glog.V(3).Infof("invalid delete volume req: %v", req)
		return nil, err
	}

	mayaConfig := mayaproxy.GetNewMayaConfig()
	err := mayaConfig.SetupMayaConfig(mayaproxy.K8sClient{})
	if err != nil {
		glog.Errorf("Error setting up MayaConfig")
		return nil, status.Error(codes.Unavailable, fmt.Sprint(err))
	}

	err = mayaConfig.DeleteVolume(req.VolumeId)
	if err != nil {
		// TODO: Handle volume delete error
	}

	return &csi.DeleteVolumeResponse{}, nil
}

func (cs *ControllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *ControllerServer) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

//TODO
func (cs *ControllerServer) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

//TODO
func (cs *ControllerServer) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ControllerGetCapabilities implements the default GRPC callout.
// Default supports all capabilities
func (cs *ControllerServer) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	glog.V(5).Infof("Using default ControllerGetCapabilities")

	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: cs.Driver.GetControllerServiceCapability(),
	}, nil
}

func (cs *ControllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	return &csi.ValidateVolumeCapabilitiesResponse{}, nil
}
