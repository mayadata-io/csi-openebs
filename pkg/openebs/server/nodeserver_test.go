package server

import (
	"testing"
	"github.com/openebs/csi-openebs/pkg/openebs/driver"
	"context"
	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/codes"
	"fmt"
)

var (
	ns = NodeServer{Driver: &driver.CSIDriver{Name: "csi-openebs", NodeID: "node-1", Version: "0.0.1"}}
)

const (
	volume1 = "csi-volume-1"
)

func init() {
	iscsiManager = ISCSIManager{iscsiService: &MockISCSIService{}}

}

type MockISCSIService struct {
	ISCSIService
}

func (util *MockISCSIService) AttachDisk(b iscsiDiskMounter) (string, error) {
	if b.targetPath == "/mount/path" {
		return "", nil
	}
	return "", errors.New(fmt.Sprintf("iscsi: failed to mkdir %s, error", b.targetPath))
}

func (util *MockISCSIService) DetachDisk(c iscsiDiskUnmounter, targetPath string) error {
	if targetPath == "/mountexist" {
		return nil

	}
	return errors.New("Warning: Unmount skipped because path does not exist: " + targetPath)
}

func TestNodePublishVolume(t *testing.T) {
	// Volume attributes with complete information
	attributesComplete := make(map[string]string)
	attributesComplete["targetPortal"] = "192.168.10.25"
	attributesComplete["iqn"] = "iqn.2016-09.com.openebs.jiva:pvc-84bbb63f-6001-11e8-8a85-42010a8e0002"
	attributesComplete["lun"] = "0"
	attributesComplete["portals"] = "[\"10.103.7.228:3260\"]"
	attributesComplete["iscsiInterface"] = "default"
	attributesComplete["initiatorName"] = "openebs-vm"

	testCases := map[string]struct {
		req *csi.NodePublishVolumeRequest
		err error
	}{
		"success": {&csi.NodePublishVolumeRequest{VolumeId: volume1,
			TargetPath: "/mount/path",
			VolumeAttributes: attributesComplete,
			Readonly: true}, nil},

		"mountFailure": {&csi.NodePublishVolumeRequest{VolumeId: volume1,
			TargetPath: "/wrong/mount/path",
			VolumeAttributes: attributesComplete,
			Readonly: true}, status.Error(codes.Internal, "iscsi: failed to mkdir /wrong/mount/path, error")},

		"missingTargetInformationFailure": {&csi.NodePublishVolumeRequest{VolumeId: volume1,
			TargetPath: "/mount/path",
			VolumeAttributes: make(map[string]string),
			Readonly: true}, status.Error(codes.Internal, "iSCSI target information is missing")},
	}

	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			_, err := ns.NodePublishVolume(context.Background(), v.req)
			assert.Equal(t, err, v.err)
		})
	}

}

func TestNodeUnpublishVolume(t *testing.T) {
	testCases := map[string]struct {
		req *csi.NodeUnpublishVolumeRequest
		err error
	}{
		"success": {&csi.NodeUnpublishVolumeRequest{TargetPath: "/mountexist", VolumeId: volume1,}, nil},
		"failure": {&csi.NodeUnpublishVolumeRequest{TargetPath: "/mountmissing", VolumeId: volume1,}, status.Error(13, "Warning: Unmount skipped because path does not exist: /mountmissing")},
	}

	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			_, err := ns.NodeUnpublishVolume(context.Background(), v.req)
			assert.Equal(t, err, v.err)
		})
	}
}

func TestNodeStageVolume(t *testing.T) {
	testCases := map[string]struct {
		req *csi.NodeStageVolumeRequest
		err error
	}{
		"failure": {&csi.NodeStageVolumeRequest{}, status.Error(codes.Unimplemented, "")},
	}

	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			_, err := ns.NodeStageVolume(context.Background(), v.req)
			assert.Equal(t, err, v.err)
		})
	}
}

func TestNodeUnstageVolume(t *testing.T) {
	testCases := map[string]struct {
		req *csi.NodeUnstageVolumeRequest
		err error
	}{
		"failure": {&csi.NodeUnstageVolumeRequest{}, status.Error(codes.Unimplemented, "")},
	}

	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			_, err := ns.NodeUnstageVolume(context.Background(), v.req)
			assert.Equal(t, err, v.err)
		})
	}
}

func
TestNodeGetId(t *testing.T) {
	testCases := map[string]struct {
		req *csi.NodeGetIdRequest
	}{
		"success": {&csi.NodeGetIdRequest{}},
	}

	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			resp, err := ns.NodeGetId(context.Background(), v.req)
			assert.Nil(t, err)
			assert.Equal(t, "node-1", resp.NodeId)
		})
	}
}

func
TestNodeGetCapabilities(t *testing.T) {
	testCases := map[string]struct {
		req *csi.NodeGetCapabilitiesRequest
	}{
		"success": {&csi.NodeGetCapabilitiesRequest{}},
	}

	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			resp, err := ns.NodeGetCapabilities(context.Background(), v.req)
			assert.Nil(t, err)
			assert.Equal(t, len(resp.Capabilities), 1)
			assert.Equal(t, csi.NodeServiceCapability_RPC_UNKNOWN, resp.Capabilities[0].GetRpc().GetType())
		})
	}
}
