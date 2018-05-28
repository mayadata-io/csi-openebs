/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"testing"
	"k8s.io/api/core/v1"
	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"fmt"
	mayav1 "github.com/openebs/csi-openebs/pkg/openebs/v1"
	"encoding/json"
	"github.com/openebs/csi-openebs/pkg/openebs/mayaproxy"
	"net/url"
	"context"
	"errors"
	"k8s.io/client-go/kubernetes"
	"strings"
)

const (
	oneGB      = "1073741824B"
	volumeList = `
{
   "items":[
      {
         "metadata":{
            "annotations":{
               "openebs.io/jiva-controller-ips":"172.17.0.9",
               "openebs.io/jiva-replica-ips":"172.17.0.8,nil,nil",
               "openebs.io/jiva-replica-status":"Running,Pending,Pending",
               "openebs.io/jiva-target-portal":"10.97.170.71:3260",
               "vsm.openebs.io/cluster-ips":"10.97.170.71",
               "vsm.openebs.io/iqn":"iqn.2016-09.com.openebs.jiva:pvc-84bbb63f-6001-11e8-8a85-42010a8e0002",
               "deployment.kubernetes.io/revision":"1",
               "openebs.io/jiva-controller-status":"Running",
               "vsm.openebs.io/replica-status":"Running,Pending,Pending",
               "vsm.openebs.io/controller-status":"Running",
               "openebs.io/storage-pool":"default",
               "openebs.io/jiva-replica-count":"3",
               "vsm.openebs.io/controller-ips":"172.17.0.9",
               "vsm.openebs.io/replica-ips":"172.17.0.8,nil,nil",
               "openebs.io/volume-monitor":"false",
               "vsm.openebs.io/replica-count":"3",
               "vsm.openebs.io/volume-size":"1073741824B",
               "openebs.io/capacity":"1073741824B",
               "vsm.openebs.io/targetportals":"10.97.170.71:3260",
               "openebs.io/jiva-controller-cluster-ip":"10.97.170.71",
               "openebs.io/jiva-iqn":"iqn.2016-09.com.openebs.jiva:pvc-84bbb63f-6001-11e8-8a85-42010a8e0002",
               "openebs.io/volume-type":"jiva"
            },
            "creationTimestamp":null,
            "labels":{

            },
            "name":"pvc-84bbb63f-6001-11e8-8a85-42010a8e0002"
         },
         "status":{
            "Message":"",
            "Phase":"",
            "Reason":""
         }
      },
      {
         "metadata":{
            "annotations":{
               "openebs.io/jiva-replica-ips":"172.17.0.7,nil,nil",
               "vsm.openebs.io/cluster-ips":"10.108.114.41",
               "vsm.openebs.io/iqn":"iqn.2016-09.com.openebs.jiva:pvc-d35a9973-5f65-11e8-8a85-42010a8e0002",
               "openebs.io/jiva-replica-count":"3",
               "vsm.openebs.io/volume-size":"1073741824B",
               "openebs.io/volume-type":"jiva",
               "vsm.openebs.io/controller-ips":"172.17.0.2",
               "vsm.openebs.io/replica-status":"Running,Pending,Pending",
               "vsm.openebs.io/targetportals":"10.108.114.41:3260",
               "deployment.kubernetes.io/revision":"1",
               "openebs.io/volume-monitor":"false",
               "vsm.openebs.io/controller-status":"Running",
               "vsm.openebs.io/replica-ips":"172.17.0.7,nil,nil",
               "openebs.io/jiva-replica-status":"Running,Pending,Pending",
               "openebs.io/jiva-target-portal":"10.108.114.41:3260",
               "vsm.openebs.io/replica-count":"3",
               "openebs.io/jiva-controller-ips":"172.17.0.2",
               "openebs.io/jiva-controller-status":"Running",
               "openebs.io/jiva-controller-cluster-ip":"10.108.114.41",
               "openebs.io/jiva-iqn":"iqn.2016-09.com.openebs.jiva:pvc-d35a9973-5f65-11e8-8a85-42010a8e0002",
               "openebs.io/storage-pool":"default",
               "openebs.io/capacity":"1073741824B"
            },
            "creationTimestamp":null,
            "labels":{

            },
            "name":"pvc-d35a9973-5f65-11e8-8a85-42010a8e0002"
         },
         "status":{
            "Message":"",
            "Phase":"",
            "Reason":""
         }
      }
   ],
   "metadata":{

   }
}`
)

var (
	// controller object to call functions
	controller ControllerServer

	// internal error from maya api server
	err error

	// vars to switch behaviour of method i.e. respond with error or no error
	countGetVolume    int
	countListVolumes  int
	countCreateVolume int
	countDeleteVolume int

	// default valid values
	mapiURI   *url.URL
	port      int32
	listenUrl string

	// vars of mocking structs
	mConfig    *mayaproxy.MayaConfig
	mService   MockMayaService
	mK8sClient MockK8sClient
	builder    MockMayaConfigBuilder
	wrapper    *mayaproxy.K8sClientWrapper
)

var (
	// array of volume annotations which is used to unmarshal into volume object
	annotation = []string{`{
		"vsm.openebs.io/iqn":            "iqn.2016-09.com.openebs.jiva:pvc-da18673b-533e-11e8-be33-000c29116015",
		"vsm.openebs.io/targetportals":  "10.103.7.228:3260",
		"openebs.io/jiva-target-portal": "10.103.7.228:3260",
		"openebs.io/capacity":           "3000000000B"
      }`,
		`{
		"vsm.openebs.io/iqn": "iqn.2016-09.com.openebs.cstor:pvc-da18673b-ds4w-11e8-be33-000c29116015"
      }`,
	}
)

// initializes to default mocking values
// this method can be called after end of every test functions
func resetToDefault() {
	countGetVolume = 0
	countListVolumes = 0
	countCreateVolume = 0

	listenUrl = "127.0.0.1"
	mapiURI, _ = url.Parse("http://" + listenUrl + ":69696")
	port = 69696

	// initialize mocking structs
	mService = MockMayaService{}
	mK8sClient = MockK8sClient{}
	mConfig = &mayaproxy.MayaConfig{MayaService: mService, MapiURI: *mapiURI, Namespace: "openebs"}
	builder = MockMayaConfigBuilder{}
	wrapper = &mayaproxy.K8sClientWrapper{ClientService: mK8sClient}

	// initialize controller server's vars with mocking structs
	clientWrapper = wrapper
	mayaConfig = mConfig
	mayaConfigBuilder = builder
}

// initial setup of mocked objects
func init() {
	resetToDefault()
	err = errors.New("HTTP Status error from maya-apiserver: Internal Server Error")
	controller = ControllerServer{}
}

// getMayaVolume will return an object of mayav1.Volume filling its Annotations with jsonMap
func getMayaVolume(jsonMap map[string]interface{}) mayav1.Volume {
	return mayav1.Volume{Metadata: struct {
		Annotations       interface{} `json:"annotations"`
		CreationTimestamp interface{} `json:"creationTimestamp"`
		Name              string      `json:"name"`
	}{Annotations: jsonMap, CreationTimestamp: "", Name: ""}}

}

// Mock struct for ClientService
type MockK8sClient struct {
	mayaproxy.K8sClientService
}

// Mock struct for MayaApiService
type MockMayaService struct {
	mayaproxy.MayaApiService
}

// Mock struct for MayaConfigBuilder
type MockMayaConfigBuilder struct {
	mayaproxy.Builder
}

func (builder MockMayaConfigBuilder) GetNewMayaConfig(clientWrapper *mayaproxy.K8sClientWrapper) (*mayaproxy.MayaConfig, error) {
	return mConfig, nil
}

// Mocked functions
func (mMayaService MockMayaService) GetVolume(mapiURI *url.URL, volumeName string) (*mayav1.Volume, error) {
	if countGetVolume > 0 {
		jsonMap := make(map[string]interface{})
		json.Unmarshal([]byte(annotation[0]), &jsonMap)
		v1 := getMayaVolume(jsonMap)
		return &v1, nil
	}
	countGetVolume++
	return nil, errors.New("HTTP Status error from maya-apiserver: Internal Server Error")

}
func (mMayaService MockMayaService) ListAllVolumes(mapiURI *url.URL) (*[]mayav1.Volume, error) {
	// first call w countListVolumes is always failure. Only subsequent calls can return volume list
	if countListVolumes > 0 {
		var volumesList mayav1.VolumeList
		json.Unmarshal([]byte(volumeList), &volumesList)
		return &volumesList.Items, nil
	}
	// below behaviour only to make it consistent w other mocked methods
	countListVolumes++
	return nil, errors.New("HTTP Status error from maya-apiserver: Internal Server Error")
}
func (mMayaService MockMayaService) CreateVolume(mapiURI *url.URL, spec mayav1.VolumeSpec) error {
	if countCreateVolume > 0 {
		return nil
	}
	countCreateVolume++
	return errors.New("http error")
}
func (mMayaService MockMayaService) DeleteVolume(mapiURI *url.URL, volumeName string) error {
	if countDeleteVolume > 0 {
		return nil
	}
	countDeleteVolume++
	return errors.New("")
}

func (mK8sClient MockK8sClient) getK8sClient() (*kubernetes.Clientset, error) {
	return &kubernetes.Clientset{}, nil
}
func (mK8sClient MockK8sClient) getSvcObject(client *kubernetes.Clientset, namespace string) (*v1.Service, error) {
	return &v1.Service{Spec: v1.ServiceSpec{ClusterIP: listenUrl, Ports: []v1.ServicePort{{Port: port, Name: "api"}}}}, nil
}

// Test functions
func TestCheckArguments(t *testing.T) {
	// The mocked method mayaproxy.CreateVolume will return error for the first call
	req := &csi.CreateVolumeRequest{Name: "csi-volume-1"}
	if err := checkArguments(req); err == nil {
		t.Errorf("Expected error when req=%v", req)
	}

	req = &csi.CreateVolumeRequest{Name: "csi-volume-1", VolumeCapabilities: []*csi.VolumeCapability{{AccessMode: nil}}}
	if err := checkArguments(req); err == nil {
		t.Errorf("Expected error when req=%v", req)
	}

	req = &csi.CreateVolumeRequest{Name: "csi-volume-1", VolumeCapabilities: []*csi.VolumeCapability{{AccessMode: nil}}, Parameters: map[string]string{"storage-class-name": "openebs"}}
	if err := checkArguments(req); err != nil {
		t.Errorf("Expected success when req=%v", req)
	}
}

func TestGetVolumeAttributes(t *testing.T) {
	expectedAttributes := map[string]string{
		"iscsiInterface": "default",
		"lun":            "0",
		"iqn":            "iqn.2016-09.com.openebs.jiva:pvc-da18673b-533e-11e8-be33-000c29116015",
		"targetPortal":   "10.103.7.228:3260",
		"portals":        "[\"10.103.7.228:3260\"]",
		"capacity":       "3000000000B",
	}

	// success case
	jsonMap := make(map[string]interface{})
	if err := json.Unmarshal([]byte(annotation[0]), &jsonMap); err != nil {
		t.Errorf(fmt.Sprintf("%s", err))
	}

	vol := getMayaVolume(jsonMap)
	attributes, err := getVolumeAttributes(&vol)
	if err != nil {
		t.Errorf("Unexpected error occurred: %s", err)
	}

	if len(attributes) != len(expectedAttributes) {
		t.Errorf("Unexpected length of attributes, expected %d got %d", len(expectedAttributes), len(attributes))
	}

	// check if v1=v2 for [k,v1] & [k,v2] maps
	for k, v := range expectedAttributes {
		if v != attributes[k] {
			t.Errorf("expected %v : %v got %v : %v", k, v, k, attributes[k])
		}
	}

	// error case when annotations are missing
	jsonMap = make(map[string]interface{})
	if err := json.Unmarshal([]byte(annotation[1]), &jsonMap); err != nil {
		t.Errorf(fmt.Sprintf("%s", err))
	}

	vol = getMayaVolume(jsonMap)
	_, err = getVolumeAttributes(&vol)
	if err == nil {
		t.Errorf("missing volume annotation should cause error")
	}

	// error case when volume passed is nil
	_, err = getVolumeAttributes(nil)
	if err == nil {
		t.Errorf("nil volume should have caused an error")
	}

}

func TestCreateVolumeSpec(t *testing.T) {
	// create request object
	req := &csi.CreateVolumeRequest{CapacityRange: &csi.CapacityRange{RequiredBytes: 1024 * 1024 * 1024},
		Name: "pvc-as88sd-a8s-das8f-as8df-dfgfd-88",
		Parameters: map[string]string{"storage-class-name": "openebs-sc"},
	}

	vSpec := createVolumeSpec(req)

	if vSpec.Metadata.Labels.Storage != oneGB {
		t.Errorf("Expected %s got %s", oneGB, vSpec.Metadata.Labels.Storage)
	}
	if vSpec.Metadata.Labels.StorageClass != "openebs-sc" {
		t.Errorf("Expected openebs-sc got %s", vSpec.Metadata.Labels.StorageClass)
	}
	if vSpec.Metadata.Name != "pvc-as88sd-a8s-das8f-as8df-dfgfd-88" {
		t.Errorf("Expected pvc-as88sd-a8s-das8f-as8df-dfgfd-88 got %s", vSpec.Metadata.Name)
	}
	if vSpec.Metadata.Labels.Namespace != "default" {
		t.Errorf("Expected default got %s", vSpec.Metadata.Labels.Namespace)
	}
	if vSpec.Kind != "PersistentVolumeClaim" {
		t.Errorf("Expected PersistentVolumeClaim got %s", vSpec.Kind)
	}
	if vSpec.APIVersion != "v1" {
		t.Errorf("Expected v1 got %s", vSpec.Kind)
	}

}

func TestCreateVolume(t *testing.T) {
	defer resetToDefault()

	req := &csi.CreateVolumeRequest{Name: "csi-volume-1",
		VolumeCapabilities: []*csi.VolumeCapability{{AccessMode: nil}},
		Parameters: map[string]string{"storage-class-name": "openebs"},
		CapacityRange: &csi.CapacityRange{RequiredBytes: 10000000000},
	}
	resp, err := controller.CreateVolume(context.Background(), req)

	if err == nil {
		t.Errorf("mocked mayaConfig should not have caused error")
	}

	resp, err = controller.CreateVolume(context.Background(), req)
	if err != nil {
		t.Errorf("Mocked mayaconfig etc. should not have caused error")
	} else {
		if resp.GetVolume().CapacityBytes == 10000000000 {
			t.Errorf("Wrong volume capacity created")
		}
	}

	mayaConfigBuilder = builder
	countGetVolume = 0
	mayaConfig = nil
	wrapper := &mayaproxy.K8sClientWrapper{ClientService: mK8sClient}
	clientWrapper = wrapper
	_, err = controller.CreateVolume(context.Background(), req)

	if mayaConfig == nil {
		t.Errorf("mayaConfig should have been initialized if empty initially")
	}

	countGetVolume = 0
	mayaConfig = mConfig
	req = &csi.CreateVolumeRequest{Name: "csi-volume-1",
		VolumeCapabilities: []*csi.VolumeCapability{{AccessMode: nil}},
		Parameters: map[string]string{"storage-class-name": "openebs"},
		CapacityRange: &csi.CapacityRange{RequiredBytes: 10000000000},
	}

	resp, err = controller.CreateVolume(context.Background(), req)
	if err != nil {
		t.Errorf("mocked mayaConfig should not have caused error")
	}
}

func TestListVolumes(t *testing.T) {
	defer resetToDefault()

	// create an empty reqest
	req := &csi.ListVolumesRequest{}
	res, err := controller.ListVolumes(context.Background(), req)
	if err == nil {
		t.Errorf("internal server error from mapi server should cause ListVolumes to fail")
	}

	// set to one explicitly
	countListVolumes = 1

	res, err = controller.ListVolumes(context.Background(), req)
	if err != nil {
		t.Errorf("ListVolume failed with correct data")
	}
	if len(res.Entries) < 2 {
		t.Errorf("expected 2 volumes got %d", len(res.Entries))
	}

	// reset to initial value
	countListVolumes = 0

}

func TestDeleteVolume(t *testing.T) {
	defer resetToDefault()
	req := &csi.DeleteVolumeRequest{}
	_, err = controller.DeleteVolume(context.Background(), req)
	if err == nil {
		t.Errorf("expected error when volume could not be deleted at mapi server")
	}

	_, err := controller.DeleteVolume(context.Background(), req)
	if err != nil {
		t.Errorf("expected no error when volume is successfully deleted at mapi server")
	}

}

func TestControllerPublishVolume(t *testing.T) {
	req := &csi.ControllerPublishVolumeRequest{}
	_, err := controller.ControllerPublishVolume(context.Background(), req)
	if err == nil || !strings.Contains(err.Error(), "Unimplemented") {
		t.Errorf("expected error 12: Unimplemented got %s", err)
	}
}

func TestControllerUnpublishVolume(t *testing.T) {
	req := &csi.ControllerUnpublishVolumeRequest{}
	_, err := controller.ControllerUnpublishVolume(context.Background(), req)
	if err == nil || !strings.Contains(err.Error(), "Unimplemented") {
		t.Errorf("expected error 12: Unimplemented got %s", err)
	}
}

func TestGetCapacity(t *testing.T) {
	req := &csi.GetCapacityRequest{}
	_, err := controller.GetCapacity(context.Background(), req)
	if err == nil || !strings.Contains(err.Error(), "Unimplemented") {
		t.Errorf("expected error 12: Unimplemented got %s", err)
	}
}
