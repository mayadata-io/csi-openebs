package mayaproxy

import (
	"net/url"
	"net/http/httptest"
	"net/http"
	"net"
)

var (
	mapiURI   *url.URL
	port      int32
	listenUrl string
	ts        *httptest.Server
)

func init() {
	// initial setup of mocked objects
	listenUrl = "127.0.0.1"
	mapiURI = &url.URL{Host: listenUrl, Scheme: "http"}
	mapiURI, _ = url.Parse("http://" + listenUrl)
	port = 69696
}

func createAndStartServer(handlerFunc http.HandlerFunc) {
	ts = httptest.NewUnstartedServer(handlerFunc)
	ts.Listener.Close()
	listener, _ := net.Listen("tcp", listenUrl)

	ts.Listener = listener
	ts.Start()
}
