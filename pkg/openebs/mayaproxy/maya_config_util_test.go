package mayaproxy

import (
	"net/url"
	"testing"
	"github.com/stretchr/testify/assert"
	"errors"
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
	testCases := map[string]struct {
		inputVersion string
		err          error
	}{
		"success": {"v2", nil},
		"failure": {"", errors.New("invalid version")},
	}
	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			err := validateVersion(v.inputVersion)
			assert.Equal(t, v.err, err)
		})
	}
}

func TestValidateVolumeName(t *testing.T) {
	testCases := map[string]struct {
		inputName string
		err       error
	}{
		"success": {"my-vol", nil},
		"failure": {"", errors.New("invalid volume name")},
	}
	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			err := validateVolumeName(v.inputName)
			assert.Equal(t, v.err, err)
		})
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
	testCases := map[string]struct {
		inputUrl                                     *url.URL
		version, volumeName, expectedDeleteVolumeUrl string
		err                                          error
	}{
		"success": {
			mcuMapiURI, versionLatest, "pvc-prince-12345",
			mcuMapiURI.String() + "/" + "latest/volumes/delete/pvc-prince-12345", nil,
		}, /*"failureEmptyVersion": {mcuMapiURI, "", "pvc-1212", "",
			errors.New("invalid version")},
		"failureEmptyVolumeName": {mcuMapiURI, "v2", "", "",
			errors.New("invalid volume name")},*/
	}

	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			volUrl, err := mayaService.GetVolumeDeleteURL(v.inputUrl, v.version, v.volumeName)
			if v.expectedDeleteVolumeUrl != "" {
				assert.Equal(t, v.expectedDeleteVolumeUrl, volUrl.String())
			}
			assert.Equal(t, v.err, err)
		})
	}
}

func TestGetVolumeInfoURL(t *testing.T) {
	testCases := map[string]struct {
		inputUrl                               *url.URL
		version, volumeName, expectedVolumeUrl string
		err                                    error
	}{
		"success": {
			mcuMapiURI, versionLatest, "pvc-1212",
			mcuMapiURI.String() + "/" + "latest/volumes/info/pvc-1212", nil,
		}, /*
		"failureEmptyVersion": {mcuMapiURI, "", "pvc-1212", "",
			errors.New("invalid version")},
		"failureEmptyVolumeName": {mcuMapiURI, "v2", "", "",
			errors.New("invalid volume name")},*/
	}

	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			volUrl, err := mayaService.GetVolumeInfoURL(v.inputUrl, v.version, v.volumeName)
			if v.expectedVolumeUrl != "" {
				assert.Equal(t, v.expectedVolumeUrl, volUrl.String())
			}
			assert.Equal(t, v.err, err)
		})
	}
}
