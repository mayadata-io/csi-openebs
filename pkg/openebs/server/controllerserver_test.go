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
	"fmt"
	mayav1 "github.com/princerachit/csi-openebs/pkg/openebs/v1"
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
	annotation := []string{`{
		"vsm.openebs.io/iqn":            "iqn.2016-09.com.openebs.jiva:pvc-da18673b-533e-11e8-be33-000c29116015",
		"vsm.openebs.io/targetportals":  "10.103.7.228:3260",
		"openebs.io/jiva-target-portal": "10.103.7.228:3260",
		"openebs.io/capacity":           "3G"
      }`,
		`{
		"vsm.openebs.io/iqn":            "iqn.2016-09.com.openebs.cstor:pvc-da18673b-ds4w-11e8-be33-000c29116015",
		"vsm.openebs.io/targetportals":  "10.103.7.228:3260",
		"openebs.io/jiva-target-portal": "10.103.7.228:3260",
		"openebs.io/capacity":           "5G"
      }`,
	}

	var expectedAttributes = make([]map[string]string, 2)

	expectedAttributes[0] = map[string]string{
		"iscsiInterface": "default",
		"lun":            "0",
		"iqn":            "iqn.2016-09.com.openebs.jiva:pvc-da18673b-533e-11e8-be33-000c29116015",
		"targetPortal":   "10.103.7.228:3260",
		"portals":        "[\"10.103.7.228:3260\"]",
		"capacity":       "3G",
	}

	expectedAttributes[1] = map[string]string{
		"iscsiInterface": "default",
		"lun":            "0",
		"iqn":            "iqn.2016-09.com.openebs.cstor:pvc-da18673b-ds4w-11e8-be33-000c29116015",
		"targetPortal":   "10.103.7.228:3260",
		"portals":        "[\"10.103.7.228:3260\"]",
		"capacity":       "5G",
	}

	for i := 0; i < 2; i++ {
		if err := json.Unmarshal([]byte(annotation[i]), &jsonMap); err != nil {
			t.Errorf(fmt.Sprintf("%s", err))
		}
		v1 := mayav1.Volume{Metadata: struct {
			Annotations       interface{} `json:"annotations"`
			CreationTimestamp interface{} `json:"creationTimestamp"`
			Name              string      `json:"name"`
		}{Annotations: jsonMap, CreationTimestamp: "", Name: ""}}

		attributes := getVolumeAttributes(&v1)

		if len(attributes) != len(expectedAttributes[i]) {
			t.Errorf("Unexpected length of attributes")
		}

		for k, v := range expectedAttributes[i] {
			if v != attributes[k] {
				t.Errorf("Expected %v : %v got %v : %v", k, v, k, attributes[k])
			}
		}
	}
}

func TestCreateVolumeSpec(t *testing.T) {
	req := &csi.CreateVolumeRequest{CapacityRange: &csi.CapacityRange{RequiredBytes: 1024 * 1024 * 1024},
		Name: "pvc-as88sd-a8s-das8f-as8df-dfgfd-88",
		Parameters: map[string]string{"storage-class-name": "openebs-sc"},
	}

	vSpec := createVolumeSpec(req)

	if vSpec.Metadata.Labels.Storage != "1G" {
		t.Errorf("Expected 1G got %s", vSpec.Metadata.Labels.Storage)
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

	req = &csi.CreateVolumeRequest{CapacityRange: &csi.CapacityRange{RequiredBytes: 1024 * 1024},
		Name: "pvc-as88sd-a8s-das8f-as8df-dfgfd-88",
		Parameters: map[string]string{"storage-class-name": "openebs-sc"},
	}

	vSpec = createVolumeSpec(req)

	if vSpec.Metadata.Labels.Storage != "0G" {
		t.Errorf("Expected 1G got %s", vSpec.Metadata.Labels.Storage)
	}
}

func TestListVolumes(t *testing.T) {
	resp := `{"items":[{"metadata":{"annotations":{"openebs.io/volume-monitor":"false","vsm.openebs.io/controller-status":"Running","openebs.io/jiva-target-portal":"10.106.196.157:3260","vsm.openebs.io/iqn":"iqn.2016-09.com.openebs.jiva:pvc-2f9de7d5-5e7c-11e8-9832-42010a8e0002","vsm.openebs.io/volume-size":"1073741824B","openebs.io/capacity":"1073741824B","openebs.io/jiva-controller-ips":"172.17.0.4","openebs.io/jiva-replica-ips":"172.17.0.8,nil,nil","vsm.openebs.io/cluster-ips":"10.106.196.157","openebs.io/volume-type":"jiva","deployment.kubernetes.io/revision":"1","openebs.io/jiva-replica-count":"3","vsm.openebs.io/controller-ips":"172.17.0.4","vsm.openebs.io/replica-status":"Running,Pending,Pending","vsm.openebs.io/targetportals":"10.106.196.157:3260","openebs.io/jiva-controller-cluster-ip":"10.106.196.157","openebs.io/jiva-iqn":"iqn.2016-09.com.openebs.jiva:pvc-2f9de7d5-5e7c-11e8-9832-42010a8e0002","openebs.io/storage-pool":"default","vsm.openebs.io/replica-count":"3","openebs.io/jiva-controller-status":"Running","vsm.openebs.io/replica-ips":"172.17.0.8,nil,nil","openebs.io/jiva-replica-status":"Running,Pending,Pending"},"creationTimestamp":null,"labels":{},"name":"pvc-2f9de7d5-5e7c-11e8-9832-42010a8e0002"},"status":{"Message":"","Phase":"","Reason":""}}],"metadata":{}}`

}
