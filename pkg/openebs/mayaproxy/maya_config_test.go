package mayaproxy

import (
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"net/url"
	"testing"
	"github.com/stretchr/testify/assert"
	"errors"
)

var (
	client     K8sClient
	mayaConfig = &MayaConfig{MapiURI: url.URL{Host: "1.1.1.1", Scheme: "http"}, Namespace: "openebs"}
)

type K8sClientMock struct {
}

func (k8sClient *K8sClientMock) getK8sClient() (*kubernetes.Clientset, error) {
	return &kubernetes.Clientset{}, nil
}

func (k8sClient *K8sClientMock) getSvcObject(client *kubernetes.Clientset, namespace string) (*v1.Service, error) {
	return &v1.Service{Spec: v1.ServiceSpec{ClusterIP: "10.20.20.30", Ports: []v1.ServicePort{{Port: 5656, Name: "api"}}}}, nil
}

func TestSetupMayaConfig(t *testing.T) {
	testCases := map[string]struct {
		client K8sClientService
		err    error
	}{
		"success": {&K8sClientMock{}, nil},
	}
	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			err := mayaConfig.SetupMayaConfig(v.client)
			assert.Nil(t, err)
		})
	}
}

func TestGetK8sClient(t *testing.T) {
	// always result in error because test cases don't run in kubernetes cluster
	_, err := client.getK8sClient()
	assert.Equal(t, errors.New("unable to load in-cluster configuration, KUBERNETES_SERVICE_HOST and KUBERNETES_SERVICE_PORT must be defined"), err)
}

func TestGetNewMayaConfig(t *testing.T) {
	testCases := map[string]struct {
		inputWrapper *K8sClientWrapper
		err          error
	}{
		"success": {&K8sClientWrapper{&K8sClientMock{}}, nil},
	}

	for k, v := range testCases {
		t.Run(k, func(t *testing.T) {
			config, err := MayaConfigBuilder{}.GetNewMayaConfig(v.inputWrapper)
			assert.Equal(t, v.err, err)
			if v.err != nil {
				assert.NotNil(t, config)
			}

		})
	}
}
