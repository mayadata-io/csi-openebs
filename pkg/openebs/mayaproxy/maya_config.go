package mayaproxy

import (
	"net/url"
	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/api/core/v1"
	mayav1 "github.com/princerachit/csi-openebs/pkg/openebs/v1"
	"os"
	"fmt"
	"errors"
)

type MayaApiService interface {
	GetMayaClusterIP(client kubernetes.Interface) (string, error)
	CreateVolume(mapiURI *url.URL, spec mayav1.VolumeSpec) error
	DeleteVolume(mapiURI *url.URL, volumeName string) error
	GetVolume(mapiURI *url.URL, volumeName string) (*mayav1.Volume, error)
	ListAllVolumes(mapiURI *url.URL) (*[]mayav1.Volume, error)
	MayaApiServerUrl
}

type K8sClientService interface {
	getK8sClient() (*kubernetes.Clientset, error)
	getSvcObject(client *kubernetes.Clientset, namespace string) (*v1.Service, error)
}

type K8sClient struct {
	K8sClientService
}

type K8sClientWrapper struct {
	k8sClient K8sClient
}

// MayaConfig is an aggregate of configurations related to mApi server
type MayaConfig struct {
	// Maya-API Server URL running in the cluster
	MapiURI url.URL
	// Namespace where openebs operator runs
	Namespace string

	MayaService MayaApiService
}

type MayaService struct {
	MayaApiService
}

func (mayaConfig MayaConfig) GetURL() *url.URL {
	return &mayaConfig.MapiURI
}

func (mayaConfig MayaConfig) GetNamespace() string {
	return mayaConfig.Namespace
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

	// setup Namespace
	if mayaConfig.Namespace = os.Getenv("OPENEBS_NAMESPACE"); mayaConfig.Namespace == "" {
		mayaConfig.Namespace = "default"
	}
	glog.Info("OpenEBS volume provisioner Namespace ", mayaConfig.Namespace)

	client, err := k8sClient.getK8sClient()
	if err != nil {
		return errors.New("Error creating kubernetes clientset")
	}

	// setup MapiURI using the Maya API Server Service
	svc, err := k8sClient.getSvcObject(client, mayaConfig.Namespace)
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
	mayaConfig.MapiURI = *mapiUrl
	glog.V(2).Infof("Maya Cluster IP: %v", mayaConfig.MapiURI)
	glog.V(2).Infof("Host: %v Scheme: %v Path: %v", mayaConfig.MapiURI.Host, mayaConfig.MapiURI.Scheme, mayaConfig.MapiURI.Path)
	return nil
}

func GetNewMayaConfig(clientWrapper *K8sClientWrapper) (*MayaConfig, error) {
	if clientWrapper == nil {
		clientWrapper = &K8sClientWrapper{k8sClient: K8sClient{}}
	}
	config := &MayaConfig{MayaService: &MayaService{}}
	err := config.SetupMayaConfig(clientWrapper.k8sClient)
	if err != nil {
		config = nil
	}
	return config, err
}
