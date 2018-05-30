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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	"github.com/openebs/csi-openebs/pkg/openebs/mayaproxy"
	mayav1 "github.com/openebs/csi-openebs/pkg/openebs/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"net/url"
	"testing"
	"net/http"
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

// initializes to default mocking structs
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
func getMayaVolume(jsonMap map[string]interface{}) *mayav1.Volume {
	return &mayav1.Volume{Metadata: struct {
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
		return v1, nil
	}
	countGetVolume++
	return nil, errors.New(http.StatusText(404))

}
func (mMayaService MockMayaService) ListAllVolumes(mapiURI *url.URL) (*[]mayav1.Volume, error) {
	// first call (default countListVolumes=0) is always failure.
	if countListVolumes > 0 {
		var volumesList mayav1.VolumeList
		json.Unmarshal([]byte(volumeList), &volumesList)
		return &volumesList.Items, nil
	}
	// To make subsequent calls work
	countListVolumes++
	return nil, errors.New("HTTP Status error from maya-apiserver: Internal Server Error")
}

func (mMayaService MockMayaService) CreateVolume(mapiURI *url.URL, spec mayav1.VolumeSpec) error {
	// first call (default countCreateVolume=0) is always failure.
	if countCreateVolume > 0 {
		return nil
	}
	// To make subsequent calls work
	countCreateVolume++
	return errors.New(http.StatusText(http.StatusInternalServerError))
}

func (mMayaService MockMayaService) DeleteVolume(mapiURI *url.URL, volumeName string) error {
	if volumeName == "csi-volume-1" {
		return nil
	}
	return errors.New("Internal Server Error")
}

func (mK8sClient MockK8sClient) getK8sClient() (*kubernetes.Clientset, error) {
	return &kubernetes.Clientset{}, nil
}
func (mK8sClient MockK8sClient) getSvcObject(client *kubernetes.Clientset, namespace string) (*v1.Service, error) {
	return &v1.Service{Spec: v1.ServiceSpec{ClusterIP: listenUrl, Ports: []v1.ServicePort{{Port: port, Name: "api"}}}}, nil
}

// Test functions
func TestCheckArguments(t *testing.T) {
	defer resetToDefault()
	testCases := map[string]struct {
		req *csi.CreateVolumeRequest
		err error
	}{
		"failureMissingVolName":          {&csi.CreateVolumeRequest{}, errors.New("missing name in request")},
		"failureMissingVolCapability":    {&csi.CreateVolumeRequest{Name: "csi-volume-1"}, errors.New("missing volume capabilities in request")},
		"failureMissingStorageClassName": {&csi.CreateVolumeRequest{Name: "csi-volume-1", VolumeCapabilities: []*csi.VolumeCapability{{AccessMode: nil}}}, errors.New("missing storage-class-name in request")},
		"success": {&csi.CreateVolumeRequest{Name: "csi-volume-1",
			VolumeCapabilities: []*csi.VolumeCapability{{AccessMode: nil}},
			Parameters: map[string]string{"storage-class-name": "openebs"}}, nil},
	}

	// run test sequentially because mocked method CreateVolume' behaviour is dependent on number of calls made
	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			assert.Equal(t, checkArguments(v.req), v.err)
		})
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

	validJsonMap := make(map[string]interface{})
	json.Unmarshal([]byte(annotation[0]), &validJsonMap)
	InvalidJsonMap := make(map[string]interface{})
	json.Unmarshal([]byte(annotation[1]), &InvalidJsonMap)

	testCases := map[string]struct {
		input          *mayav1.Volume
		expectedOutput map[string]string
		err            error
	}{
		"success":            {getMayaVolume(validJsonMap), expectedAttributes, nil},
		"failureAnnotation":  {getMayaVolume(InvalidJsonMap), nil, errors.New("required volume attribute " + mayav1.OpenebsTargetPortal + " be cannot nil")},
		"failureEmptyVolume": {nil, nil, errors.New("volume or its annotations cannot be nil")},
	}
	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			attributes, err := getVolumeAttributes(v.input)
			if v.expectedOutput != nil {
				for key, val := range expectedAttributes {
					assert.Equal(t, val, attributes[key])
				}
			}
			assert.Equal(t, err, v.err)
		})
	}
}

func TestCreateVolumeSpec(t *testing.T) {
	testCases := map[string]struct {
		req                                                      *csi.CreateVolumeRequest
		storage, storageClass, name, namespace, kind, apiVersion string
	}{
		"success": {&csi.CreateVolumeRequest{CapacityRange: &csi.CapacityRange{RequiredBytes: 1024 * 1024 * 1024}, Name: "pvc-as88sd-a8s-das8f-as8df-dfgfd-88", Parameters: map[string]string{"storage-class-name": "openebs-sc"},},
			oneGB, "openebs-sc", "pvc-as88sd-a8s-das8f-as8df-dfgfd-88", "default", "PersistentVolumeClaim", "v1"},
	}
	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			vSpec := createVolumeSpec(v.req)
			assert.Equal(t, v.storage, vSpec.Metadata.Labels.Storage)
			assert.Equal(t, v.storageClass, vSpec.Metadata.Labels.StorageClass)
			assert.Equal(t, v.name, vSpec.Metadata.Name)
			assert.Equal(t, v.namespace, vSpec.Metadata.Labels.Namespace)
			assert.Equal(t, v.kind, vSpec.Kind)
			assert.Equal(t, v.apiVersion, vSpec.APIVersion)
		})
	}
}

func TestCreateVolume(t *testing.T) {
	defer resetToDefault()
	validRequest := &csi.CreateVolumeRequest{Name: "csi-volume-1",
		VolumeCapabilities: []*csi.VolumeCapability{{AccessMode: nil}},
		Parameters: map[string]string{"storage-class-name": "openebs"},
		CapacityRange: &csi.CapacityRange{RequiredBytes: 3000000000},
	}

	testCases := map[string]struct {
		req  *csi.CreateVolumeRequest
		resp *csi.CreateVolumeResponse
		err  error
	}{
		"failure": {validRequest, nil, status.Error(codes.Unavailable, fmt.Sprintf("Error from maya-api-server: %s", http.StatusText(http.StatusInternalServerError)))},
		"success": {validRequest, &csi.CreateVolumeResponse{Volume: &csi.Volume{Id: "csi-volume-1", CapacityBytes: 3000000000}}, nil},
	}

	// run test sequentially because mocked method CreateVolume' behaviour is dependent on number of calls made
	for k, v := range testCases {
		if k == "success" {
			// if countGetVolume zero only then mocked CreateVolume method will called
			countGetVolume = 0
		}
		resp, err := controller.CreateVolume(context.Background(), v.req)
		assert.Equal(t, v.err, err)
		if v.resp != nil {
			assert.Equal(t, v.resp.Volume.GetCapacityBytes(), resp.Volume.GetCapacityBytes())
		}
	}
}

func TestListVolumes(t *testing.T) {
	defer resetToDefault()

	testCases := map[string]struct {
		req *csi.ListVolumesRequest
		len int
		err error
	}{
		"failure": {&csi.ListVolumesRequest{}, 0, status.Error(codes.Unavailable, "HTTP Status error from maya-apiserver: Internal Server Error")},
		"success": {&csi.ListVolumesRequest{}, 2, nil},
	}

	// run test sequentially because mocked method ListAllVolumes' behaviour is dependent on number of calls made
	for _, v := range testCases {
		resp, err := controller.ListVolumes(context.Background(), v.req)
		assert.Equal(t, v.err, err)
		if v.len > 0 {
			assert.Equal(t, v.len, len(resp.Entries))
		}
	}
}

func TestDeleteVolume(t *testing.T) {
	defer resetToDefault()

	testCases := map[string]struct {
		req  *csi.DeleteVolumeRequest
		resp *csi.DeleteVolumeResponse
		err  error
	}{
		"success": {&csi.DeleteVolumeRequest{VolumeId: "csi-volume-1"}, &csi.DeleteVolumeResponse{}, nil},
		"failure": {&csi.DeleteVolumeRequest{VolumeId: "csi-volume-2"}, nil, status.Error(codes.Unavailable, "Error from maya-api-server: "+http.StatusText(http.StatusInternalServerError))},
	}

	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			resp, err := controller.DeleteVolume(context.Background(), v.req)
			assert.Equal(t, v.err, err)
			assert.Equal(t, v.resp, resp)
		})
	}
}

func TestControllerPublishVolume(t *testing.T) {
	testCases := map[string]struct {
		req *csi.ControllerPublishVolumeRequest
	}{
		"success": {&csi.ControllerPublishVolumeRequest{}},
	}

	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			resp, err := controller.ControllerPublishVolume(context.Background(), v.req)
			assert.Nil(t, resp)
			assert.Error(t, status.Error(codes.Unimplemented, ""), err)
		})
	}
}

func TestControllerUnpublishVolume(t *testing.T) {
	testCases := map[string]struct {
		req *csi.ControllerUnpublishVolumeRequest
	}{
		"success": {&csi.ControllerUnpublishVolumeRequest{}},
	}

	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			resp, err := controller.ControllerUnpublishVolume(context.Background(), v.req)
			assert.Nil(t, resp)
			assert.Error(t, status.Error(codes.Unimplemented, ""), err)
		})
	}
}

func TestGetCapacity(t *testing.T) {
	testCases := map[string]struct {
		req *csi.GetCapacityRequest
	}{
		"success": {&csi.GetCapacityRequest{}},
	}

	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			resp, err := controller.GetCapacity(context.Background(), v.req)
			assert.Nil(t, resp)
			assert.Error(t, status.Error(codes.Unimplemented, ""), err)
		})
	}
}
