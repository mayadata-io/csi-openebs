package server

import (
	"testing"
	"context"
	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"github.com/openebs/csi-openebs/pkg/openebs/driver"
)

var (
	ids = &IdentityServer{Driver: &driver.CSIDriver{Name: "csi-openebs"}}
)

func TestGetPluginInfo(t *testing.T) {
	pluginInfo, err := ids.GetPluginInfo(context.Background(), &csi.GetPluginInfoRequest{})
	if pluginInfo == nil || err != nil {
		t.Errorf("Error retrieving plugin info")
	}
	if pluginInfo.Name != "csi-openebs" {
		t.Errorf("expected plugin name 'csi-openebs' got %s", pluginInfo.Name)
	}
}

func TestProbe(t *testing.T) {
	probe, err := ids.Probe(context.Background(), &csi.ProbeRequest{})
	if probe == nil || err != nil {
		t.Errorf("Error in probing driver")
	}
}

func TestGetPluginCapabilities(t *testing.T) {
	cap, err := ids.GetPluginCapabilities(context.Background(), &csi.GetPluginCapabilitiesRequest{})
	if err != nil {
		t.Errorf("Error in retrieving plugin capabilities")
	}
	if len(cap.GetCapabilities()) != 1 && cap.GetCapabilities()[0].GetService().Type != csi.PluginCapability_Service_CONTROLLER_SERVICE {
		t.Errorf("Unwxpected plugin capabilities")
	}
}
