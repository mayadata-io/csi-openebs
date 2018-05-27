package mayaproxy

import (
	"net/url"
	"net/http/httptest"
	"net/http"
	"net"
	"testing"
	mayav1 "github.com/princerachit/csi-openebs/pkg/openebs/v1"
	"io/ioutil"
	"strings"
)

var (
	mpMapiURI *url.URL
	ts        *httptest.Server
	spec1     *mayav1.VolumeSpec
	spec2     *mayav1.VolumeSpec
	mService  *MayaService
)

const (
	listenUrl = "127.0.0.1"
	portStr   = "6966"

	volume1           = "csi-volume-1"
	volume2           = "csi-volume-2"
	volume3           = "csi-volume-3"
	getVolumeResponse = `{
		"vsm.openebs.io/iqn":            "iqn.2016-09.com.openebs.jiva:pvc-da18673b-533e-11e8-be33-000c29116015",
		"vsm.openebs.io/targetportals":  "10.103.7.228:3260",
		"openebs.io/jiva-target-portal": "10.103.7.228:3260",
		"openebs.io/capacity":           "3000000000B"
      }`
	successResponseBodyCreateVolume = `{
    "metadata": {
        "creationTimestamp": null,
        "labels": {},
        "name": "csi-volume-1"
    },
    "status": {
        "Message": "",
        "Phase": "",
        "Reason": ""
    }
	}`

	listVolumesResponse = `{
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

func initServerURI() {
	mpMapiURI = &url.URL{Host: listenUrl + ":" + portStr, Scheme: "http"}
}

// initial setup of mocked objects
func init() {
	initServerURI()

	mService = &MayaService{}

	metadata1 := mayav1.VolumeSpec{}.Metadata
	metadata1.Name = "csi-volume-1"
	spec1 = &mayav1.VolumeSpec{Metadata: metadata1}

	metadata2 := mayav1.VolumeSpec{}.Metadata
	metadata2.Name = "csi-volume-2"
	spec2 = &mayav1.VolumeSpec{Metadata: metadata2}
}

// createAndStartServer creates a test server with given handler and starts it
func createAndStartServer(handlerFunc http.HandlerFunc) {
	ts = nil
	ts = httptest.NewUnstartedServer(handlerFunc)
	ts.Listener.Close()
	listener, err := net.Listen("tcp", listenUrl+":"+portStr)
	if err != nil {
		panic("Listener invalid")
	}
	ts.Listener = listener
	ts.Start()
}

// tearDownServer is a cleanup function to stop test server
func tearDownServer() {
	ts.Close()
	ts.Listener.Close()
}

func TestCreateVolume(t *testing.T) {

	// Request handler
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			if r.RequestURI == "/latest/volumes/" {
				b, _ := ioutil.ReadAll(r.Body)
				requestBody := string(b)
				// success when volume name is csi-volume-1
				if strings.Contains(requestBody, volume1) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(successResponseBodyCreateVolume))
					// failure when volume name is csi-volume-2
				} else if strings.Contains(requestBody, volume2) {
					w.WriteHeader(http.StatusInternalServerError)
				}
			}
		}
	}

	createAndStartServer(handler)
	defer tearDownServer()

	// Volume successfully created
	err := mService.CreateVolume(mpMapiURI, *spec1)
	if err != nil {
		t.Errorf("Volume creation failed")
	}

	// Server error
	err = mService.CreateVolume(mpMapiURI, *spec2)
	if err == nil {
		t.Errorf("Internal server error should cause volume creation failure")
	}

}

func TestDeleteVolume(t *testing.T) {

	// Request handler
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			if r.RequestURI == "/latest/volumes/delete/"+volume1 {
				w.WriteHeader(http.StatusOK)
			} else if r.RequestURI == "/latest/volumes/delete/"+volume2 {
				w.WriteHeader(http.StatusInternalServerError)
			} else if r.RequestURI == "/latest/volumes/delete/"+volume3 {
				w.WriteHeader(http.StatusNotFound)
			}

		}
	}

	createAndStartServer(handler)
	defer tearDownServer()

	// Volume successfully created
	err := mService.DeleteVolume(mpMapiURI, volume1)
	if err != nil {
		t.Errorf("Volume deletion failed")
	}

	err = mService.DeleteVolume(mpMapiURI, volume2)
	if err == nil {
		t.Errorf("Internal error should cause deletion failure")
	}

	err = mService.DeleteVolume(mpMapiURI, volume3)
	if err != nil {
		t.Errorf("Volume deletion should not fail for already deleted volume")
	}
}

func TestRequestServerGet(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			if r.RequestURI == "/success" {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusBadGateway)
			}
		}
	}

	createAndStartServer(handler)
	defer tearDownServer()

	_, err := requestServerGet(mpMapiURI)
	if err == nil {
		t.Errorf("Error from mapi server should cause error")
	}

	mpMapiURI.Path = "success"
	defer initServerURI()

	_, err = requestServerGet(mpMapiURI)
	if err != nil {
		t.Errorf("200 Response from server should not cause error")
	}
}

func TestGetVolume(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			if r.RequestURI == "/latest/volumes/info/"+volume1 {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(getVolumeResponse))
			} else if r.RequestURI == "/latest/volumes/info/"+volume2 {
				w.WriteHeader(http.StatusBadGateway)
			} else if r.RequestURI == "/latest/volumes/info/"+volume3 {
				w.WriteHeader(http.StatusNotFound)
			}
		}
	}

	createAndStartServer(handler)
	defer tearDownServer()

	vol, err := mService.GetVolume(mpMapiURI, volume1)
	if err != nil {
		t.Errorf("volume response from server should not cause error")
	}
	if vol == nil {
		t.Errorf("volume response from server should not result in empty volume object")
	}

	_, err = mService.GetVolume(mpMapiURI, volume2)
	if err == nil {
		t.Errorf("server error should cause an error")
	}

	_, err = mService.GetVolume(mpMapiURI, volume3)
	if err == nil {
		t.Errorf("server error should cause an error")
	}
}

func TestListVolumesResponse(t *testing.T) {
	// TODO: Add test case when volumes don't exist

	// success with volumes list
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			if r.RequestURI == "/latest/volumes/" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(listVolumesResponse))
			}
		}
	}

	createAndStartServer(handler)
	volumes, err := mService.ListAllVolumes(mpMapiURI)
	if err != nil {
		t.Errorf("List volume error")
	}
	if len(*volumes) != 2 {
		t.Errorf("Expected 2 volumes got %d", len(*volumes))
	}
	tearDownServer()

	handler = func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			if r.RequestURI == "/latest/volumes/" {
				w.WriteHeader(http.StatusInternalServerError)
			}
		}
	}

	createAndStartServer(handler)
	defer tearDownServer()

	_, err = mService.ListAllVolumes(mpMapiURI)
	if err == nil {
		t.Errorf("server error should have caused error")
	}
}
