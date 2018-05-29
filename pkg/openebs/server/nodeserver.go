/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANYcon KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/openebs/csi-openebs/pkg/openebs/driver"
	"github.com/golang/glog"
)

// NodeServer implements csi.NodeServer interface
type NodeServer struct {
	csi.NodeServer
	Driver *driver.CSIDriver
}

type ISCSIManager struct {
	iscsiService ISCSIService
}

type ISCSIService interface {
	AttachDisk(b iscsiDiskMounter) (string, error)
	DetachDisk(c iscsiDiskUnmounter, targetPath string) error
}

var (
	iscsiManager = ISCSIManager{iscsiService: &ISCSIUtil{}}
)

// NodePublishVolume publishes the openebs volume.
func (ns *NodeServer) NodePublishVolume(ctx context.Context, req *csi.NodePublishVolumeRequest) (*csi.NodePublishVolumeResponse, error) {
	iscsiInfo, err := getISCSIInfo(req)
	glog.V(4).Infof("iscsiInfo: %v", iscsiInfo)
	if err != nil {
		glog.Errorf("Failed to get iscsiInfo: %s", err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	diskMounter := getISCSIDiskMounter(iscsiInfo, req)
	glog.V(4).Infof("diskMounter: %v", diskMounter)
	_, err = iscsiManager.iscsiService.AttachDisk(*diskMounter)
	if err != nil {
		glog.Errorf("Failed to attach disk: %s", err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &csi.NodePublishVolumeResponse{}, nil
}

// NodeUnpublishVolume unpublishes the openebs volume.
func (ns *NodeServer) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	diskUnmounter := getISCSIDiskUnmounter(req)
	glog.V(4).Infof("diskUnmounter: %v", diskUnmounter)
	targetPath := req.GetTargetPath()

	err := iscsiManager.iscsiService.DetachDisk(*diskUnmounter, targetPath)
	if err != nil {
		glog.Errorf("Failed to detach disk from at %s: %s", targetPath, err)
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &csi.NodeUnpublishVolumeResponse{}, nil
}

// NodeStageVolume is unimplemented
func (ns *NodeServer) NodeStageVolume(ctx context.Context, req *csi.NodeStageVolumeRequest) (*csi.NodeStageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// NodeUnstageVolume is unimplemented
func (ns *NodeServer) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
}

// NodeGetId returns the node ID
func (ns *NodeServer) NodeGetId(ctx context.Context, req *csi.NodeGetIdRequest) (*csi.NodeGetIdResponse, error) {
	return &csi.NodeGetIdResponse{NodeId: ns.Driver.NodeID,}, nil
}

// NodeGetCapabilities returns unknown capability
func (ns *NodeServer) NodeGetCapabilities(ctx context.Context, req *csi.NodeGetCapabilitiesRequest) (*csi.NodeGetCapabilitiesResponse, error) {
	return &csi.NodeGetCapabilitiesResponse{
		Capabilities: []*csi.NodeServiceCapability{
			{
				Type: &csi.NodeServiceCapability_Rpc{
					Rpc: &csi.NodeServiceCapability_RPC{
						Type: csi.NodeServiceCapability_RPC_UNKNOWN,
					},
				},
			},
		},
	}, nil
}
