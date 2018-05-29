package mayaproxy

import (
	"encoding/json"
	"github.com/golang/glog"
	"net/http"
	"bytes"
	"io/ioutil"
	mayav1 "github.com/openebs/csi-openebs/pkg/openebs/v1"
	"gopkg.in/yaml.v2"
	"errors"
	"net/url"
)

// CreateVolume requests mapi server to create an openebs volume. It returns an error if volume creation fails
func (mayaService *MayaService) CreateVolume(mapiURI *url.URL, spec mayav1.VolumeSpec) error {
	// Marshal serializes the value provided into a YAML document
	yamlValue, _ := yaml.Marshal(spec)

	glog.V(3).Infof("Volume spec yaml created:\n%v\n", string(yamlValue))

	url, err := mayaService.GetVolumeURL(mapiURI, versionLatest)
	glog.V(3).Infof("Create volume URL %v", url.String())
	if err != nil {
		return err
	}

	glog.V(4).Infof("Requesting maya-api-server to create volume")
	req, err := http.NewRequest("POST", url.String(), bytes.NewBuffer(yamlValue))
	req.Header.Add("Content-Type", "application/yaml")
	c := &http.Client{
		Timeout: timeout,
	}

	resp, err := c.Do(req)
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	code := resp.StatusCode
	if code != http.StatusOK {
		return errors.New(http.StatusText(code))
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Errorf("unable to read response from maya-api-server %s", err)
		return err
	}

	responseBody := string(data)
	glog.V(3).Infof("volume successfully created:\n%v\n", responseBody)
	return nil
}

// DeleteVolume requests mapi server to delete an openebs volume. It returns an error if volume deletion fails
func (mayaService *MayaService) DeleteVolume(mapiURI *url.URL, volumeName string) error {
	url, err := mayaService.GetVolumeDeleteURL(mapiURI, versionLatest, volumeName)
	if err != nil {
		return err
	}

	glog.V(3).Infof("Requesting for volume delete at %s", url.String())
	resp, err := requestServerGet(url)
	defer resp.Body.Close()
	if err != nil {
		// if volume does not exist then no problem. It has been deleted in previous calls.
		if err.Error() == http.StatusText(404) {
			return nil
		}
		return err
	}

	glog.Infof("Volume deletion successfully initiated for %s", volumeName)
	return nil
}

// GetVolume requests mapi server to GET the details of a volume and returns it by filling into *mayav1.Volume.
// If the volume does not exist or can't be retrieved then it returns an error
func (mayaService *MayaService) GetVolume(mapiURI *url.URL, volumeName string) (*mayav1.Volume, error) {
	var volume mayav1.Volume

	url, err := mayaService.GetVolumeInfoURL(mapiURI, versionLatest, volumeName)
	if err != nil {
		return nil, err
	}
	glog.V(3).Infof("Requesting for volume details at %s", url.String())

	resp, err := requestServerGet(url)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}
	// Fill the obtained json into volume
	json.NewDecoder(resp.Body).Decode(&volume)
	glog.Infof("Volume details successfully retrieved %v", volume)

	return &volume, nil
}

// ListAllVolumes requests mapi server to GET the details of all volumes
func (mayaService *MayaService) ListAllVolumes(mapiURI *url.URL) (*[]mayav1.Volume, error) {
	var volumesList mayav1.VolumeList

	url, err := mayaService.GetVolumeURL(mapiURI, versionLatest)
	if err != nil {
		return nil, err
	}
	glog.V(3).Infof("Requesting for volumes list at %s", url.String())

	resp, err := requestServerGet(url)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}
	// Fill the obtained json into volume
	json.NewDecoder(resp.Body).Decode(&volumesList)
	glog.Infof("volume Details Successfully Retrieved %v", volumesList)

	return &volumesList.Items, nil
}

// requestServerGet performs a get request to the given url. It returns error even if response code is not 200
func requestServerGet(url *url.URL) (*http.Response, error) {
	req, err := http.NewRequest("GET", url.String(), nil)
	c := &http.Client{
		Timeout: timeout,
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	code := resp.StatusCode
	if code != http.StatusOK {
		return nil, errors.New(http.StatusText(code))
	}

	return resp, nil
}
