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
	mcuMapiURI = &url.URL{
		Scheme: urlScheme,
		Host:   mApiUrl,
	}
	mayaService = &MayaService{}
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

	obtainedUrl, err := mayaService.GetVolumeURL(mcuMapiURI, versionLatest)
	if err != nil {
		t.Error(err)
	}
	expectedVolumeUrl := mcuMapiURI.String() + "/" + "latest/volumes/"

	if obtainedUrl.String() != expectedVolumeUrl {
		t.Errorf("Expected %s got %s", expectedVolumeUrl, obtainedUrl.String())
	}
	obtainedUrl, err = mayaService.GetVolumeURL(mcuMapiURI, "")
	if err == nil {
		t.Error("Empty version should cause an error")
	}

	obtainedUrl, err = mayaService.GetVolumeURL(mcuMapiURI, "latest")
	if obtainedUrl.String() != expectedVolumeUrl {
		t.Errorf("Expected %s got %s", expectedVolumeUrl, obtainedUrl.String())
	}
}

func TestGetVolumeDeleteURL(t *testing.T) {

	obtainedUrl, err := mayaService.GetVolumeDeleteURL(mcuMapiURI, versionLatest, "pvc-prince-12345")
	if err != nil {
		t.Error(err)
	}
	expectedVolumeUrl := mcuMapiURI.String() + "/" + "latest/volumes/delete/pvc-prince-12345"

	if obtainedUrl.String() != expectedVolumeUrl {
		t.Errorf("Expected %s got %s", expectedVolumeUrl, obtainedUrl.String())
	}

	obtainedUrl, err = mayaService.GetVolumeDeleteURL(mcuMapiURI, "", "pvc-prince-12345")
	if err == nil {
		t.Error("Empty version should cause an error")
	}

	mcuMapiURI.Scheme = ""
	defer func() { mcuMapiURI.Scheme = urlScheme }()

	obtainedUrl, err = mayaService.GetVolumeDeleteURL(mcuMapiURI, "", "pvc-prince-12345")
	if err == nil {
		t.Error("Empty volume name should cause an error")
	}

}

func TestGetVolumeInfoURL(t *testing.T) {

	obtainedUrl, err := mayaService.GetVolumeInfoURL(mcuMapiURI, versionLatest, "pvc-1212")
	if err != nil {
		t.Error(err)
	}
	expectedVolumeUrl := mcuMapiURI.String() + "/" + "latest/volumes/info/pvc-1212"

	if obtainedUrl.String() != expectedVolumeUrl {
		t.Errorf("Expected %s got %s", expectedVolumeUrl, obtainedUrl.String())
	}

	obtainedUrl, err = mayaService.GetVolumeInfoURL(mcuMapiURI, "", "pvc-1212")
	if err == nil {
		t.Error("Empty version should cause an error")
	}

	obtainedUrl, err = mayaService.GetVolumeInfoURL(mcuMapiURI, "v2", "")
	if err == nil {
		t.Error("Empty volume name should cause an error")
	}
}
