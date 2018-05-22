package mayaproxy

import (
	"net/url"
	"strings"
	"errors"
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
	timeout = 60 * time.Second
)

type MayaApiServerUrl interface {
	GetVolumeURL(version string) (url.URL, error)
	GetVolumeDeleteURL(version string) (url.URL, error)
	GetVolumeInfoURL(version, volumeName string) (url.URL, error)
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
func (mayaConfig MayaConfig) GetVolumeURL(version string) (*url.URL, error) {
	err := validateVersion(version)
	if err != nil {
		return nil, err
	}
	scheme := mayaConfig.mapiURI.Scheme
	if scheme == "" {
		scheme = "http"
	}
	return &url.URL{
		Scheme: scheme,
		Host:   mayaConfig.mapiURI.Host,
		Path:   "/" + version + "/" + volumes + "/",
	}, nil
}

// GetVolumeDeleteURL returns the volume url of mApi server
func (mayaConfig MayaConfig) GetVolumeDeleteURL(version, volumeName string) (*url.URL, error) {
	deleteVolumeURL, err := mayaConfig.GetVolumeURL(version)
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
func (mayaConfig MayaConfig) GetVolumeInfoURL(version, volumeName string) (*url.URL, error) {
	volumeInfoURL, err := mayaConfig.GetVolumeURL(version)
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
