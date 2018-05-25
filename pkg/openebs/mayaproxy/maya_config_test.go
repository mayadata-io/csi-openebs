package mayaproxy

import (
	"testing"
	"k8s.io/client-go/kubernetes"
	"k8s.io/api/core/v1"
	"net/url"
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
	mayaConfig := &MayaConfig{}

	err := mayaConfig.SetupMayaConfig(&K8sClientMock{})
	if err != nil {
		t.Errorf("Test should have passed for ClusterIp: 10.20.20.30 and an empty client set")
	}

}

func TestGetK8sClient(t *testing.T) {
	_, err := client.getK8sClient()
	if err == nil {
		t.Errorf("Missing kubernetes server should have caused error")
	}
}

func TestGetNamespace(t *testing.T) {
	if mayaConfig.Namespace != mayaConfig.GetNamespace() {
		t.Errorf("Wrong Namespace returned")
	}
}

func TestGetNewMayaConfig(t *testing.T) {
	_, err := MayaConfigBuilder{}.GetNewMayaConfig(&K8sClientWrapper{&K8sClientMock{}})
	if err != nil {
		t.Errorf("Mocked Client Service should not have caused error")
	}

	config, err := MayaConfigBuilder{}.GetNewMayaConfig(nil)
	if err == nil {
		t.Errorf("Missing kubernetes server should have caused error")
	}
	if config != nil {
		t.Errorf("MayaConfig should be nil if setup fails")
	}
}
