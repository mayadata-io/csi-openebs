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
)

const (
	listenUrl = "127.0.0.1"
	port      = 6966
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
)

func initServerURI() {
	mpMapiURI = &url.URL{Host: listenUrl + ":" + portStr, Scheme: "http"}
}

// initial setup of mocked objects
func init() {
	initServerURI()

	metadata1 := mayav1.VolumeSpec{}.Metadata
	metadata1.Name = "csi-volume-1"
	spec1 = &mayav1.VolumeSpec{Metadata: metadata1}

	metadata2 := mayav1.VolumeSpec{}.Metadata
	metadata2.Name = "csi-volume-2"
	spec2 = &mayav1.VolumeSpec{Metadata: metadata2}
}

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

	mService := &MayaService{}

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

	mService := &MayaService{}

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

func TestReqVolume(t *testing.T) {
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

	_, err := reqVolume(mpMapiURI)
	if err == nil {
		t.Errorf("Error from mapi server should cause error")
	}

	mpMapiURI.Path = "success"
	defer initServerURI()

	_, err = reqVolume(mpMapiURI)
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

	mService := &MayaService{}
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
