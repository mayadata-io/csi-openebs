package mayaproxy

import (
	"encoding/json"
	"github.com/golang/glog"
	"net/http"
	"bytes"
	"io/ioutil"
	mayav1 "github.com/princerachit/csi-openebs/pkg/openebs/v1"
	"gopkg.in/yaml.v2"
	"errors"
	"net/url"
)

// CreateVolume requests mapi server to create an openebs volume. It returns an error if volume creation fails
func (mayaService *MayaService) CreateVolume(mapiURI *url.URL, spec mayav1.VolumeSpec) error {
	// Marshal serializes the value provided into a YAML document
	yamlValue, _ := yaml.Marshal(spec)

	glog.Infof("[DEBUG] volume Spec Created:\n%v\n", string(yamlValue))

	url, err := mayaService.GetVolumeURL(mapiURI, versionLatest)
	glog.Infof("[DEBUG] create volume URL %v", url.String())
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url.String(), bytes.NewBuffer(yamlValue))
	req.Header.Add("Content-Type", "application/yaml")
	c := &http.Client{
		Timeout: timeout,
	}
	resp, err := c.Do(req)

	if err != nil {
		glog.Errorf("Error when connecting maya-apiserver %v", err)
		return err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Errorf("Unable to read response from maya-apiserver %v", err)
		return err
	}
	code := resp.StatusCode
	if code != http.StatusOK {
		glog.Errorf("Error response from maya-apiserver: %v", http.StatusText(code))
		return errors.New("error response from maya-apiserver")
	}

	responseBody := string(data)
	glog.Infof("volume Successfully Created:\n%v\n", responseBody)
	return nil
}

// DeleteVolume requests mapi server to delete an openebs volume. It returns an error if volume deletion fails
func (mayaService *MayaService) DeleteVolume(mapiURI *url.URL, volumeName string) error {
	glog.Infof("[DEBUG] Delete Volume :%v", string(volumeName))

	url, err := mayaService.GetVolumeDeleteURL(mapiURI, versionLatest, volumeName)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	c := &http.Client{
		Timeout: timeout,
	}
	resp, err := c.Do(req)

	if err != nil {
		glog.Errorf("Error when connecting to maya-apiserver  %v", err)
		return err
	}
	defer resp.Body.Close()

	code := resp.StatusCode
	if code != http.StatusOK {
		// is it not found? then no worry. Volume is already deleted
		if code == 404 {
			return nil
		}
		glog.Errorf("HTTP Status error from maya-apiserver: %v\n", http.StatusText(code))
		return errors.New(http.StatusText(code))
	}
	glog.Info("volume Deletion Successfully initiated")
	return nil
}

// GetVolume requests mapi server to GET the details
// of a volume and returns it by filling into *mayav1.Volume.
// If the volume does not exist or can't be retrieved then it returns an error
func (mayaService *MayaService) GetVolume(mapiURI *url.URL, volumeName string) (*mayav1.Volume, error) {
	var volume mayav1.Volume

	url, err := mayaService.GetVolumeInfoURL(mapiURI, versionLatest, volumeName)
	if err != nil {
		return nil, err
	}
	glog.Infof("[DEBUG] Requesting for volume details at %s", url.String())

	resp, err := reqVolume(url)
	// What to do if error is 404 i.e volume does not exist?
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// Fill the obtained json into volume
	json.NewDecoder(resp.Body).Decode(&volume)
	glog.Infof("volume Details Successfully Retrieved %v", volume)

	return &volume, nil
}

// ListAllVolumes requests mapi server to GET the details of all volumes
func (mayaService *MayaService) ListAllVolumes(mapiURI *url.URL) (*[]mayav1.Volume, error) {
	var volumesList mayav1.VolumeList

	glog.Infof("[DEBUG] Get details for all volumesList")

	url, err := mayaService.GetVolumeURL(mapiURI, versionLatest)
	if err != nil {
		return nil, err
	}
	glog.Infof("[DEBUG] Request Url %s", url)
	resp, err := reqVolume(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// Fill the obtained json into volume
	json.NewDecoder(resp.Body).Decode(&volumesList)
	glog.Infof("volume Details Successfully Retrieved %v", volumesList)

	return &volumesList.Items, nil
}

func reqVolume(url *url.URL) (*http.Response, error) {
	req, err := http.NewRequest("GET", url.String(), nil)
	c := &http.Client{
		Timeout: timeout,
	}

	resp, err := c.Do(req)
	if err != nil {
		glog.Errorf("Error when connecting to maya-apiserver %v", err)
		return nil, err
	}

	code := resp.StatusCode
	if code != http.StatusOK {
		glog.Errorf("HTTP Status error from maya-apiserver: %v\n", http.StatusText(code))
		return nil, errors.New("HTTP Status error from maya-apiserver: " + http.StatusText(code))
	}

	return resp, nil
}
