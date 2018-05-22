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
	"github.com/container-storage-interface/spec/lib/go/csi/v0"
	mayav1 "github.com/kubernetes-incubator/external-storage/openebs/types/v1"
	"fmt"
	"encoding/json"
)

func TestCheckArguments(t *testing.T) {
	req := &csi.CreateVolumeRequest{Name: "csi-volume-1"}
	fmt.Printf("Test 1: req=%v", req)
	if err := checkArguments(req); err == nil {
		t.Errorf("Expected error when req=%v", req)
	} else {
		fmt.Println("passed\n")
	}

	req = &csi.CreateVolumeRequest{Name: "csi-volume-1", VolumeCapabilities: []*csi.VolumeCapability{&csi.VolumeCapability{AccessMode: nil}}}
	fmt.Printf("Test 2: req=%v", req)
	if err := checkArguments(req); err == nil {
		t.Errorf("Expected error when req=%v", req)
	} else {
		fmt.Println("passed\n")
	}

	req = &csi.CreateVolumeRequest{Name: "csi-volume-1", VolumeCapabilities: []*csi.VolumeCapability{&csi.VolumeCapability{AccessMode: nil}}, Parameters: map[string]string{"storage-class-name": "openebs"}}
	fmt.Printf("Test 3: req=%v", req)
	if err := checkArguments(req); err != nil {
		t.Errorf("Expected success when req=%v", req)
	} else {
		fmt.Println("passed\n")
	}
}

func TestGetVolumeAttributes(t *testing.T) {
	jsonMap := make(map[string]interface{})
	var annotation = `{
		"vsm.openebs.io/iqn":            "iqn.2016-09.com.openebs.jiva:pvc-da18673b-533e-11e8-be33-000c29116015",
		"vsm.openebs.io/targetportals":  "10.103.7.228:3260",
		"openebs.io/jiva-target-portal": "10.103.7.228:3260"
	}`
	if err := json.Unmarshal([]byte(annotation), &jsonMap); err != nil {
		t.Errorf(fmt.Sprintf("%s", err))
	}
	v1 := mayav1.Volume{Metadata: struct {
		Annotations       interface{} `json:"annotations"`
		CreationTimestamp interface{} `json:"creationTimestamp"`
		Name              string      `json:"name"`
	}{Annotations: jsonMap, CreationTimestamp: "", Name: ""}}

	attributes := getVolumeAttributes(&v1)

	if len(attributes) != 5 {
		t.Errorf("Unexpected length of attributes")
	}

	expectedAttributes := map[string]string{"iscsiInterface": "default", "lun": "0", "iqn": "iqn.2016-09.com.openebs.jiva:pvc-da18673b-533e-11e8-be33-000c29116015", "targetPortal": "10.103.7.228:3260", "portals": "10.103.7.228:3260"}

	for k, v := range expectedAttributes {
		if v != attributes[k] {
			t.Errorf("Expected %v : %v got %v : %v", k, v, k, attributes[k])
		}
	}
}

func TestCreateVolumeSpec(t *testing.T) {

}
