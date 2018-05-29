package server

import (
	"testing"
	"github.com/openebs/csi-openebs/pkg/openebs/driver"
	"context"
	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"errors"
)

var (
	ns = NodeServer{Driver: &driver.CSIDriver{Name: "csi-openebs", NodeID: "node-1", Version: "0.0.1"}}
)

const (
	volume1 = "csi-volume-1"
	volume2 = "csi-volume-2"
)

func init() {
	iscsiManager = ISCSIManager{iscsiService: &MockISCSIService{}}
}

type MockISCSIService struct {
	ISCSIService
}

func (util *MockISCSIService) AttachDisk(b iscsiDiskMounter) (string, error) {
	if b.VolName == "csi-volume-1" {
		return "", nil
	}
	return "", errors.New("error in mounting path")
}

func (util *MockISCSIService) DetachDisk(c iscsiDiskUnmounter, targetPath string) error {
	if targetPath == "/mountexist" {
		return nil

	}
	return errors.New("targetPath " + targetPath + "path not found")
}

func TestNodePublishVolume(t *testing.T) {
	attributes := make(map[string]string)
	attributes["targetPortal"] = "192.168.10.25"
	attributes["iqn"] = "iqn.2016-09.com.openebs.jiva:pvc-84bbb63f-6001-11e8-8a85-42010a8e0002"
	attributes["lun"] = "0"
	attributes["portals"] = "[\"10.103.7.228:3260\"]"
	attributes["iscsiInterface"] = "default"
	attributes["initiatorName"] = "openebs-vm"

	_, err := ns.NodePublishVolume(context.Background(), &csi.NodePublishVolumeRequest{VolumeId: volume1,
		TargetPath: "/mount/path",
		VolumeAttributes: attributes,
		Readonly: true})

	if err != nil {
		t.Errorf("error in volume publish")
	}

	_, err = ns.NodePublishVolume(context.Background(), &csi.NodePublishVolumeRequest{VolumeId: volume2,
		TargetPath: "/mount/path",
		VolumeAttributes: attributes,
		Readonly: true})

	if err == nil {
		t.Errorf("mount failure should cause volume publish call failure")
	}

	// remove target portal
	delete(attributes, "targetPortal")

	_, err = ns.NodePublishVolume(context.Background(), &csi.NodePublishVolumeRequest{VolumeId: volume1,
		TargetPath: "/mount/path",
		VolumeAttributes: attributes,
		Readonly: true})

	if err == nil {
		t.Errorf("missing targetPortal should cause volume publish failure")
	}

}

func TestNodeUnpublishVolume(t *testing.T) {
	nodeUnpublishVolumeRequest := &csi.NodeUnpublishVolumeRequest{TargetPath: "/mountexist", VolumeId: volume1,}
	_, err := ns.NodeUnpublishVolume(context.Background(), nodeUnpublishVolumeRequest)

	if err != nil {
		t.Errorf("volume unpublish failed")
	}

	nodeUnpublishVolumeRequest = &csi.NodeUnpublishVolumeRequest{TargetPath: "/mountmissing", VolumeId: volume1,}
	_, err = ns.NodeUnpublishVolume(context.Background(), nodeUnpublishVolumeRequest)

	if err == nil {
		t.Errorf("volume unpublish failure should cause failure")
	}
}

func TestNodeStageVolume(t *testing.T) {
	_, err := ns.NodeStageVolume(context.Background(), &csi.NodeStageVolumeRequest{})
	if err != nil {
		t.Errorf("nodeStageVolume should not return error")
	}
}

func TestNodeUnstageVolume(t *testing.T) {
	_, err := ns.NodeUnstageVolume(context.Background(), &csi.NodeUnstageVolumeRequest{})
	if err != nil {
		t.Errorf("nodeUnstageVolume should not return error")
	}
}

func TestNodeGetId(t *testing.T) {
	resp, err := ns.NodeGetId(context.Background(), &csi.NodeGetIdRequest{})
	if err != nil {
		t.Errorf("error in getting node ID")
	}

	if resp.NodeId != "node-1" {
		t.Errorf("wrong node ID")
	}
}

func TestNodeNodeGetCapabilities(t *testing.T) {
	resp, err := ns.NodeGetCapabilities(context.Background(), &csi.NodeGetCapabilitiesRequest{})
	if err != nil {
		t.Errorf("error in getting node ID")
	}

	if len(resp.Capabilities) != 1 || resp.Capabilities[0].GetRpc().GetType() != csi.NodeServiceCapability_RPC_UNKNOWN {
		t.Errorf("wrong node capabilities")
	}
}
