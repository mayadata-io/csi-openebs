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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"github.com/golang/glog"
	"github.com/openebs/csi-openebs/pkg/openebs/driver"
	"github.com/openebs/csi-openebs/pkg/openebs/mayaproxy"
	mayav1 "github.com/openebs/csi-openebs/pkg/openebs/v1"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"strconv"
	"strings"
)

var (
	mayaConfig        *mayaproxy.MayaConfig
	clientWrapper     *mayaproxy.K8sClientWrapper
	mayaConfigBuilder mayaproxy.Builder
)

// ControllerServer implements csi.ControllerServer interface
type ControllerServer struct {
	csi.ControllerServer
	Driver *driver.CSIDriver
}

// checkArguments validates CreateVolumeRequest
func checkArguments(req *csi.CreateVolumeRequest) error {
	if len(req.GetName()) == 0 {
		return errors.New("missing name in request")
	}
	if req.GetVolumeCapabilities() == nil {
		return errors.New("missing volume capabilities in request")
	}
	if req.Parameters[mayav1.StorageClassName] == "" {
		return errors.New("missing storage-class-name in request")
	}
	return nil
}

// getVolumeAttributes iterates over volume's annotation and returns a map of attributes
func getVolumeAttributes(volume *mayav1.Volume) (map[string]string, error) {
	if volume == nil || volume.Metadata.Annotations == nil {
		return nil, errors.New("volume or its annotations cannot be nil")
	}

	var iqn, targetPortal, portals, capacity string
	var portalList []string
	annotations := volume.Metadata.Annotations.(map[string]interface{})

	volAttributes := []string{mayav1.OpenebsIqn, mayav1.OpenebsTargetPortal, mayav1.OpenebsPortals, mayav1.OpenebsCapacity}

	// check for missing annotations
	for _, attr := range volAttributes {
		if annotations[attr] == nil {
			return nil, errors.New("required volume attribute " + attr + " be cannot nil")
		}
	}

	iqn = annotations[mayav1.OpenebsIqn].(string)
	targetPortal = annotations[mayav1.OpenebsTargetPortal].(string)
	portalList = []string{annotations[mayav1.OpenebsPortals].(string)}
	capacity = annotations[mayav1.OpenebsCapacity].(string)
	marshaledPortalList, _ := json.Marshal(portalList) // marshal portal list so iscsi_util.go can unmarshal it later

	portals = string(marshaledPortalList)
	// values hardcoded below. Do they need fix?
	attributes := map[string]string{
		mayav1.Iqn:            iqn,
		mayav1.TargetPortal:   targetPortal, mayav1.Lun: "0",
		mayav1.Portals:        portals,
		mayav1.IscsiInterface: "default",
		mayav1.Capacity:       capacity,
	}
	return attributes, nil
}

// createVolumeSpec returns a volume spec created from the req object
func createVolumeSpec(req *csi.CreateVolumeRequest) mayav1.VolumeSpec {
	volumeSpec := mayav1.VolumeSpec{}

	volumeSpec.Kind = mayav1.PersistentVolumeClaim
	volumeSpec.APIVersion = "v1"
	// define size in bytes to avoid complex conversion logic
	volumeSpec.Metadata.Labels.Storage = fmt.Sprintf("%dB", req.GetCapacityRange().GetRequiredBytes())
	volumeSpec.Metadata.Labels.StorageClass = req.Parameters[mayav1.StorageClassName]
	volumeSpec.Metadata.Name = req.Name
	if req.Parameters["namespace"] == "" {
		volumeSpec.Metadata.Labels.Namespace = mayav1.DefaultNamespace
	} else {
		volumeSpec.Metadata.Labels.Namespace = req.Parameters["namespace"]
	}
	return volumeSpec
}

// setupPrecondition initializes mayaConfigBuilder and mayaConfig if not initialized yet
func setupPrecondition() error {
	glog.V(4).Infof("Initializing mayaConfig")
	var err error
	if mayaConfigBuilder == nil {
		mayaConfigBuilder = mayaproxy.MayaConfigBuilder{}
	}

	if mayaConfig == nil {
		mayaConfig, err = mayaConfigBuilder.GetNewMayaConfig(clientWrapper)
		if err != nil {
			return err
		}
	}
	return nil
}

// CreateVolume creates an openebs volume
func (cs *ControllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	glog.V(4).Infof("Validating CreateVolumeRequest")
	// Check arguments
	err := checkArguments(req)
	if err != nil {
		glog.Infof("Error in validating CreateVolumeRequest: ", err)
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// initialize mayaConfig if not initialized yet
	err = setupPrecondition()
	if err != nil {
		glog.Errorf("Initializing mayaConfig failed with error %s", err)
		return nil, status.Error(codes.Unavailable, fmt.Sprint(err))
	}

	var volume *mayav1.Volume

	glog.Infof("Pre creation check if volume %s exists", req.GetName())
	volume, err = mayaConfig.MayaService.GetVolume(&mayaConfig.MapiURI, req.GetName())
	if err != nil {
		glog.Errorf("Error in getting volume %s details", req.GetName())
		// If volume does not exist then create volume. Otherwise return error
		if err.Error() == http.StatusText(404) {
			glog.Infof("Volume %s does not exist", req.GetName())

			glog.V(3).Infof("Creating volume spec")
			volumeSpec := createVolumeSpec(req)
			glog.V(3).Infof("Volume spec created %v", volumeSpec)

			glog.Infof("Attempting to create volume")
			err = mayaConfig.MayaService.CreateVolume(&mayaConfig.MapiURI, volumeSpec)
			if err != nil {
				glog.Errorf("Error from maya-api-server: %s", err)
				return nil, status.Error(codes.Unavailable, fmt.Sprintf("Error from maya-api-server: %s", err))
			}
			glog.Infof("Volume created")

			// Fetch the details of created volume
			glog.Infof("Fetching details of volume %s", req.GetName())
			volume, err = mayaConfig.MayaService.GetVolume(&mayaConfig.MapiURI, req.GetName())
			if err != nil {
				glog.Errorf("Error fetching the created volume details from maya-api-server: %s", err)
				return nil, status.Error(codes.Unavailable, fmt.Sprintf("Error fetching the created volume details from maya-api-server: %v", err))
			}
		} else {
			glog.Errorf("Error from maya-api-server: %s", err)
			return nil, status.Error(codes.Unavailable, fmt.Sprintf("Error from maya-api-server: %s", err))
		}
	}

	glog.V(3).Infof("%s volume details:\n %v", req.Name, volume)
	glog.V(3).Infof("Extracting volume attributes")
	attributes, err := getVolumeAttributes(volume)
	if err != nil {
		glog.Errorf("Extracting volume attributes failed: %s", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("openEBS volume error: %s", err))
	}
	glog.V(4).Infof("Volume attributes %v", attributes)

	// Extract volume size
	capacity, err := strconv.ParseInt(strings.Split(attributes[mayav1.Capacity], "B")[0], 10, 64)
	if err != nil {
		// This situation should never occur ideally
		glog.Errorf("Invalid capacity '%s' volume found", attributes[mayav1.Capacity])
	}

	// if volume created size differs.
	// TODO: add more validation checks and move to different function
	if capacity != req.GetCapacityRange().GetRequiredBytes() {
		glog.Errorf("Capacity mismatch for volume %s. Want %dB has %dB", req.Name, req.GetCapacityRange().GetRequiredBytes(), capacity)
		return nil, status.Error(codes.AlreadyExists, fmt.Sprintf("Capacity mismatch for volume %s. Want %vB has %vB", req.Name, req.GetCapacityRange().GetRequiredBytes(), capacity))
	}

	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			Id:            volume.Metadata.Name,
			CapacityBytes: capacity,
			Attributes:    attributes,
		},
	}, nil
}

// DeleteVolume deletes given openebs volume
func (cs *ControllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	var err error

	err = setupPrecondition()
	if err != nil {
		glog.Errorf("Initializing mayaConfig failed with error %s", err)
		return nil, status.Error(codes.Unavailable, fmt.Sprint(err))
	}

	glog.Infof("Attempting to delete volume", string(req.VolumeId))
	err = mayaConfig.MayaService.DeleteVolume(&mayaConfig.MapiURI, req.VolumeId)
	if err != nil {
		glog.Errorf("Error from maya-api-server: %s", err)
		return nil, status.Error(codes.Unavailable, fmt.Sprintf("Error from maya-api-server: %s", err))
	}

	return &csi.DeleteVolumeResponse{}, nil
}

// ControllerPublishVolume is unimplemented
func (cs *ControllerServer) ControllerPublishVolume(ctx context.Context, req *csi.ControllerPublishVolumeRequest) (*csi.ControllerPublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ControllerUnpublishVolume is unimplemented
func (cs *ControllerServer) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ListVolumes lists all openebs volumes
func (cs *ControllerServer) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	var err error
	err = setupPrecondition()
	if err != nil {
		glog.Errorf("Initializing mayaConfig failed with error %s", err)
		return nil, status.Error(codes.Unavailable, fmt.Sprint(err))
	}
	volumes, err := mayaConfig.MayaService.ListAllVolumes(&mayaConfig.MapiURI)
	if err != nil {
		glog.Errorf("Error from maya-api-server: %s", err)
		return nil, status.Error(codes.Unavailable, fmt.Sprint(err))
	}

	var entries []*csi.ListVolumesResponse_Entry
	for _, volume := range *volumes {
		attributes, err := getVolumeAttributes(&volume)
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("openEBS volume error: %s", err))
		}
		glog.V(4).Infof("attributes %v", attributes)
		capacity, err := strconv.ParseInt(strings.Split(attributes[mayav1.Capacity], "B")[0], 10, 64)
		if err != nil {
			glog.Errorf("Invalid capacity '%s' volume found", capacity)
		}
		entries = append(entries, &csi.ListVolumesResponse_Entry{Volume: &csi.Volume{Attributes: attributes, CapacityBytes: capacity, Id: volume.Metadata.Name}})
	}
	return &csi.ListVolumesResponse{Entries: entries}, nil
}

// GetCapacity is unimplemented
func (cs *ControllerServer) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// ControllerGetCapabilities returns controller capabilities
func (cs *ControllerServer) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: cs.Driver.GetControllerServiceCapability(),
	}, nil
}

// ValidateVolumeCapabilities is used to validate volume's capabilities
func (cs *ControllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	for _, vCap := range req.VolumeCapabilities {
		if vCap.AccessMode.GetMode() != csi.VolumeCapability_AccessMode_SINGLE_NODE_WRITER {
			return &csi.ValidateVolumeCapabilitiesResponse{Supported: false, Message: ""}, nil
		}
	}
	return &csi.ValidateVolumeCapabilitiesResponse{Supported: true, Message: ""}, nil
}
