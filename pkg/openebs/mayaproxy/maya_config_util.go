package mayaproxy

import (
	"errors"
	"net/url"
	"strings"
	"time"
)

const (
	// maya api server versions
	versionLatest = "latest"

	// maya api server host path constants
	volumes = "volumes"
	delete  = "delete"
	info    = "info"

	// mapi service name
	mayaApiServerService = "maya-apiserver-service"

	// default timeout value
	timeout = 10 * time.Second
)

type MayaApiServerUrl interface {
	GetVolumeURL(mapiURI *url.URL, version string) (*url.URL, error)
	GetVolumeDeleteURL(mapiURI *url.URL, version, volumeName string) (*url.URL, error)
	GetVolumeInfoURL(mapiURI *url.URL, version, volumeName string) (*url.URL, error)
}

// returns error  if version is empty.
func validateVersion(version string) error {
	//TODO: Add more checks
	if strings.TrimSpace(version) == "" {
		return errors.New("invalid version")
	}
	return nil
}

func validateVolumeName(volumeName string) error {
	//TODO: Add more checks
	if strings.TrimSpace(volumeName) == "" {
		return errors.New("invalid volume name")
	}
	return nil
}

// GetVolumeURL returns the volume url of mApi server
func (mayaService MayaService) GetVolumeURL(mapiURI *url.URL, version string) (*url.URL, error) {
	err := validateVersion(version)
	if err != nil {
		return nil, err
	}
	scheme := mapiURI.Scheme
	if scheme == "" {
		scheme = "http"
	}
	return &url.URL{
		Scheme: scheme,
		Host:   mapiURI.Host,
		Path:   "/" + version + "/" + volumes + "/",
	}, nil
}

// GetVolumeDeleteURL returns the volume url of mApi server
func (mayaService MayaService) GetVolumeDeleteURL(mapiURI *url.URL, version, volumeName string) (*url.URL, error) {
	deleteVolumeURL, err := mayaService.GetVolumeURL(mapiURI, version)
	if err != nil {
		return nil, err
	}
	err = validateVolumeName(volumeName)
	if err != nil {
		return nil, err
	}
	deleteVolumeURL.Path = deleteVolumeURL.Path + delete + "/" + volumeName
	return deleteVolumeURL, nil
}

// GetVolumeInfoURL returns the info url for a volume
func (mayaService MayaService) GetVolumeInfoURL(mapiURI *url.URL, version, volumeName string) (*url.URL, error) {
	volumeInfoURL, err := mayaService.GetVolumeURL(mapiURI, version)
	if err != nil {
		return nil, err
	}
	err = validateVolumeName(volumeName)
	if err != nil {
		return nil, err
	}
	volumeInfoURL.Path = volumeInfoURL.Path + info + "/" + volumeName
	return volumeInfoURL, nil
}
