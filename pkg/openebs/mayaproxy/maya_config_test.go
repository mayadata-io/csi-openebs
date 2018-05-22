package mayaproxy

import (
	"testing"
	"k8s.io/client-go/kubernetes"
	"k8s.io/api/core/v1"
	"github.com/stretchr/testify/mock"
)

type K8sClientMock struct {
	mock.Mock
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
