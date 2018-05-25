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
	mayav1 "github.com/princerachit/csi-openebs/pkg/openebs/v1"
	"github.com/golang/glog"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/codes"
	"fmt"
	"encoding/json"
	"github.com/princerachit/csi-openebs/pkg/openebs/driver"
	"strconv"
	"strings"
	"errors"
)

var (
	mayaConfig        *mayaproxy.MayaConfig
	clientWrapper     *mayaproxy.K8sClientWrapper
	mayaConfigBuilder mayaproxy.Builder
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
			return nil, errors.New(fmt.Sprintf("required volume attribute %s be cannot nil", attr))
		}
	}

	iqn = annotations[mayav1.OpenebsIqn].(string)
	targetPortal = annotations[mayav1.OpenebsTargetPortal].(string)
	portalList = []string{annotations[mayav1.OpenebsPortals].(string)}
	capacity = annotations[mayav1.OpenebsCapacity].(string)
	marshaledPortalList, _ := json.Marshal(portalList) // marshal portal list so iscsi_util can unmarshal it later

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
func createVolumeSpec(req *csi.CreateVolumeRequest) (mayav1.VolumeSpec) {
	volumeSpec := mayav1.VolumeSpec{}

	// define size in bytes to avoid complex conversion logic
	volumeSpec.Kind = "PersistentVolumeClaim"
	volumeSpec.APIVersion = "v1"
	volumeSpec.Metadata.Labels.Storage = fmt.Sprintf("%dB", req.GetCapacityRange().GetRequiredBytes())
	volumeSpec.Metadata.Labels.StorageClass = req.Parameters["storage-class-name"]
	volumeSpec.Metadata.Name = req.Name
	volumeSpec.Metadata.Labels.Namespace = "default"

	return volumeSpec
}

// setupPrecondition initializes mayaConfigBuilder and mayaConfig if not initialized yet
func setupPrecondition() error {
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

func (cs *ControllerServer) CreateVolume(ctx context.Context, req *csi.CreateVolumeRequest) (*csi.CreateVolumeResponse, error) {
	// Check arguments
	err := checkArguments(req)
	if err != nil {
		return nil, err
	}

	// initialize mayaConfig if not initialized yet
	err = setupPrecondition()
	if err != nil {
		return nil, status.Error(codes.Unavailable, fmt.Sprint(err))
	}

	var volume *mayav1.Volume

	// If volume retrieval fails then create the volume
	volume, err = mayaConfig.MayaService.GetVolume(mayaConfig.GetURL(), req.GetName())
	glog.Infof("[DEBUG] Volume details get volume initially %s", volume)
	if err != nil {
		volumeSpec := createVolumeSpec(req)

		glog.Infof("Attempting to create volume")
		err = mayaConfig.MayaService.CreateVolume(mayaConfig.GetURL(), volumeSpec)

		if err != nil {
			return nil, status.Error(codes.Unavailable, fmt.Sprint(err))
		}
	}

	volume, err = mayaConfig.MayaService.GetVolume(mayaConfig.GetURL(), req.GetName())
	if err != nil {
		return nil, status.Error(codes.DeadlineExceeded, fmt.Sprintf("Unable to contact mapi server: %v", err))
	}

	glog.Infof("[DEBUG] Volume details %s", volume)
	glog.Infof("[DEBUG] Volume metadata %v", volume.Metadata)

	attributes, err := getVolumeAttributes(volume)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("openEBS volume error: %s", err))
	}
	glog.Infof("attributes %v", attributes)

	// Extract volume size
	capacity, err := strconv.ParseInt(strings.Split(attributes[mayav1.Capacity], "B")[0], 10, 64)
	if err != nil {
		glog.Errorf("invalid capacity '%s' volume found", capacity)
	}

	return &csi.CreateVolumeResponse{
		Volume: &csi.Volume{
			Id:            volume.Metadata.Name,
			CapacityBytes: capacity,
			Attributes:    attributes,
		},
	}, nil
}

func (cs *ControllerServer) DeleteVolume(ctx context.Context, req *csi.DeleteVolumeRequest) (*csi.DeleteVolumeResponse, error) {
	glog.Infof("Received request: %v", req)
	var err error

	err = setupPrecondition()
	if err != nil {
		return nil, status.Error(codes.Unavailable, fmt.Sprint(err))
	}

	err = mayaConfig.MayaService.DeleteVolume(mayaConfig.GetURL(), req.VolumeId)
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

func (cs *ControllerServer) ListVolumes(ctx context.Context, req *csi.ListVolumesRequest) (*csi.ListVolumesResponse, error) {
	glog.Infof("List Volumes req received")

	var err error
	err = setupPrecondition()
	if err != nil {
		return nil, status.Error(codes.Unavailable, fmt.Sprint(err))
	}
	volumes, err := mayaConfig.MayaService.ListAllVolumes(mayaConfig.GetURL())
	if err != nil {
		return nil, status.Error(codes.Unavailable, fmt.Sprint(err))
	}

	var entries []*csi.ListVolumesResponse_Entry
	for _, volume := range *volumes {
		attributes, err := getVolumeAttributes(&volume)
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("openEBS volume error: %s", err))
		}
		glog.Infof("attributes %v", attributes)
		capacity, err := strconv.ParseInt(strings.Split(attributes[mayav1.Capacity], "B")[0], 10, 64)
		if err != nil {
			glog.Errorf("Invalid capacity '%s' volume found", capacity)
		}
		entries = append(entries, &csi.ListVolumesResponse_Entry{Volume: &csi.Volume{Attributes: attributes, CapacityBytes: capacity, Id: volume.Metadata.Name,}})
	}
	return &csi.ListVolumesResponse{Entries: entries}, nil
}

func (cs *ControllerServer) GetCapacity(ctx context.Context, req *csi.GetCapacityRequest) (*csi.GetCapacityResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

func (cs *ControllerServer) ControllerGetCapabilities(ctx context.Context, req *csi.ControllerGetCapabilitiesRequest) (*csi.ControllerGetCapabilitiesResponse, error) {
	return &csi.ControllerGetCapabilitiesResponse{
		Capabilities: cs.Driver.GetControllerServiceCapability(),
	}, nil
}

// TODO
func (cs *ControllerServer) ValidateVolumeCapabilities(ctx context.Context, req *csi.ValidateVolumeCapabilitiesRequest) (*csi.ValidateVolumeCapabilitiesResponse, error) {
	return &csi.ValidateVolumeCapabilitiesResponse{}, nil
}
