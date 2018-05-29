package server

import (
	"testing"
	"context"
	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"github.com/openebs/csi-openebs/pkg/openebs/driver"
	"github.com/stretchr/testify/assert"
)

var (
	ids = &IdentityServer{Driver: &driver.CSIDriver{Name: "csi-openebs", Version: "0.0.1"}}
)

func TestGetPluginInfo(t *testing.T) {
	testCases := map[string]struct {
		req  *csi.GetPluginInfoRequest
		resp *csi.GetPluginInfoResponse
		err  error
	}{
		"success": {&csi.GetPluginInfoRequest{}, &csi.GetPluginInfoResponse{Name: "csi-openebs", VendorVersion: "0.0.1"}, nil},
	}

	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			resp, err := ids.GetPluginInfo(context.Background(), v.req)
			assert.Equal(t, v.err, err)
			assert.Equal(t, v.resp.Name, resp.Name)
			assert.Equal(t, v.resp.VendorVersion, resp.VendorVersion)
		})
	}
}

func TestProbe(t *testing.T) {
	testCases := map[string]struct {
		req  *csi.ProbeRequest
		resp *csi.ProbeResponse
		err  error
	}{
		"success": {&csi.ProbeRequest{}, &csi.ProbeResponse{}, nil},
	}

	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			resp, err := ids.Probe(context.Background(), v.req)
			assert.Equal(t, v.err, err)
			assert.NotNil(t, resp)
		})
	}
}

func TestGetPluginCapabilities(t *testing.T) {
	testCases := map[string]struct {
		req            *csi.GetPluginCapabilitiesRequest
		capabilityType csi.PluginCapability_Service_Type
	}{
		"success": {&csi.GetPluginCapabilitiesRequest{}, csi.PluginCapability_Service_Type(1)},
	}

	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			resp, _ := ids.GetPluginCapabilities(context.Background(), v.req)
			assert.Equal(t, v.capabilityType, resp.GetCapabilities()[0].GetService().Type)
		})
	}
}
