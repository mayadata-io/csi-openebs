package mayaproxy

import (
	"testing"
	"net/url"
)

const (
	urlScheme = "https"
	mApiUrl   = "default.svc.cluster.local:5656"
)

var (
	mayaConfig = &MayaConfig{
		MapiURI: url.URL{
			Scheme: urlScheme,
			Host:   mApiUrl,
		},
	}
)

func TestValidateVersion(t *testing.T) {
	version := ""
	err := validateVersion(version)
	if err == nil {
		t.Errorf("version value: \"%v\" should cause an error", version)
	}

	version = "v1"
	err = validateVersion(version)
	if err != nil {
		t.Errorf("version value: \"%v\" should not cause an error", version)
	}
}

func TestValidateVolumeName(t *testing.T) {
	volumeName := ""
	err := validateVolumeName(volumeName)
	if err == nil {
		t.Errorf("volume name: \"%v\" should cause an error", volumeName)
	}

	volumeName = "v1"
	err = validateVolumeName(volumeName)
	if err != nil {
		t.Errorf("volume name: \"%v\" should not cause an error", volumeName)
	}
}

func TestGetVolumeURL(t *testing.T) {

	obtainedUrl, err := mayaConfig.GetVolumeURL(versionLatest)
	if err != nil {
		t.Error(err)
	}
	expectedVolumeUrl := mayaConfig.MapiURI.String() + "/" + "latest/volumes/"

	if obtainedUrl.String() != expectedVolumeUrl {
		t.Errorf("Expected %s got %s", expectedVolumeUrl, obtainedUrl.String())
	}
	obtainedUrl, err = mayaConfig.GetVolumeURL("")
	if err == nil {
		t.Error("Empty version should cause an error")
	}
}

func TestGetVolumeDeleteURL(t *testing.T) {

	obtainedUrl, err := mayaConfig.GetVolumeDeleteURL(versionLatest, "pvc-prince-12345")
	if err != nil {
		t.Error(err)
	}
	expectedVolumeUrl := mayaConfig.MapiURI.String() + "/" + "latest/volumes/delete/pvc-prince-12345"

	if obtainedUrl.String() != expectedVolumeUrl {
		t.Errorf("Expected %s got %s", expectedVolumeUrl, obtainedUrl.String())
	}

	obtainedUrl, err = mayaConfig.GetVolumeDeleteURL("", "pvc-prince-12345")
	if err == nil {
		t.Error("Empty version should cause an error")
	}

	obtainedUrl, err = mayaConfig.GetVolumeDeleteURL("", "pvc-prince-12345")
	if err == nil {
		t.Error("Empty volume name should cause an error")
	}

}

func TestGetVolumeInfoURL(t *testing.T) {

	obtainedUrl, err := mayaConfig.GetVolumeInfoURL(versionLatest, "pvc-1212")
	if err != nil {
		t.Error(err)
	}
	expectedVolumeUrl := mayaConfig.MapiURI.String() + "/" + "latest/volumes/info/pvc-1212"

	if obtainedUrl.String() != expectedVolumeUrl {
		t.Errorf("Expected %s got %s", expectedVolumeUrl, obtainedUrl.String())
	}

	obtainedUrl, err = mayaConfig.GetVolumeInfoURL("", "pvc-1212")
	if err == nil {
		t.Error("Empty version should cause an error")
	}

	obtainedUrl, err = mayaConfig.GetVolumeInfoURL("v2", "")
	if err == nil {
		t.Error("Empty volume name should cause an error")
	}
}
