package mayaproxy

import (
	"net/url"
	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/api/core/v1"
	"os"
	"fmt"
	"errors"
)

type MayaApiService interface {
	GetMayaClusterIP(client kubernetes.Interface) (string, error)
	ProxyCreateVolume() (error)
	ProxyDeleteVolume() (error)
	MayaApiServerUrl
}

type K8sClientService interface {
	getK8sClient() (*kubernetes.Clientset, error)
	getSvcObject(client *kubernetes.Clientset, namespace string) (*v1.Service, error)
}

type K8sClient struct{}

// MayaConfig is an aggregate of configurations related to mApi server
type MayaConfig struct {
	// Maya-API Server URL running in the cluster
	mapiURI url.URL
	// namespace where openebs operator runs
	namespace string

	MayaApiService
}

func (k8sClient K8sClient) getK8sClient() (*kubernetes.Clientset, error) {
	// Create an InClusterConfig and use it to create a client for the controller
	// to use to communicate with Kubernetes
	config, err := rest.InClusterConfig()
	if err != nil {
		glog.Errorf("Failed to create config: %v", err)
		return nil, err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Errorf("Failed to create client: %v", err)
		return nil, err
	}

	return client, nil
}

func (k8sClient K8sClient) getSvcObject(client *kubernetes.Clientset, namespace string) (*v1.Service, error) {
	return client.CoreV1().Services(namespace).Get(mayaApiServerService, metav1.GetOptions{})
}

/*
Using a pointer receiver. This reduces chances of object tossing
from one method to another and makes testing easier.The callee
must check for error before proceeding to use MayaConfig
*/
func (mayaConfig *MayaConfig) SetupMayaConfig(k8sClient K8sClientService) error {

	// setup namespace
	if mayaConfig.namespace = os.Getenv("OPENEBS_NAMESPACE"); mayaConfig.namespace == "" {
		mayaConfig.namespace = "default"
	}
	glog.Info("OpenEBS volume provisioner namespace ", mayaConfig.namespace)

	client, err := k8sClient.getK8sClient()
	if err != nil {
		return errors.New("Error creating kubernetes clientset")
	}

	// setup mapiURI using the Maya API Server Service
	svc, err := k8sClient.getSvcObject(client, mayaConfig.namespace)
	if err != nil {
		glog.Errorf("Error getting maya-apiserver IP Address: %v", err)
		return errors.New("Error creating kubernetes clientset")
	}

	// Remove the hardcoding of port index
	mapiUrl, err := url.Parse("http://" + svc.Spec.ClusterIP + ":" + fmt.Sprintf("%d", svc.Spec.Ports[0].Port))
	glog.V(2).Infof("Maya apiserver spec %v", svc)
	if err != nil {
		glog.Errorf("Could not parse maya-apiserver server url: %v", err)
		return err
	}
	mayaConfig.mapiURI = *mapiUrl
	glog.V(2).Infof("Maya Cluster IP: %v", mayaConfig.mapiURI)
	glog.V(2).Infof("Host: %v Scheme: %v Path: %v", mayaConfig.mapiURI.Host, mayaConfig.mapiURI.Scheme, mayaConfig.mapiURI.Path)
	return nil
}

func GetNewMayaConfig() *MayaConfig {
	return &MayaConfig{}
}
