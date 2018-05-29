package mayaproxy

import (
	"net/url"
	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/api/core/v1"
	mayav1 "github.com/openebs/csi-openebs/pkg/openebs/v1"
	"os"
	"fmt"
)

// logic is separated into interfaces which makes testing easier
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

type Builder interface {
	GetNewMayaConfig(clientWrapper *K8sClientWrapper) (*MayaConfig, error)
}

type MayaConfigBuilder struct {
	Builder
}

type K8sClient struct {
	K8sClientService
}

type K8sClientWrapper struct {
	ClientService K8sClientService
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

// getK8sClient create an InClusterConfig and use it to create a client for the controller to use to communicate with Kubernetes
func (k8sClient K8sClient) getK8sClient() (*kubernetes.Clientset, error) {
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

// getSvcObject is a wrapper which uses kubernetes functions to get a k8s service object
func (k8sClient K8sClient) getSvcObject(client *kubernetes.Clientset, namespace string) (*v1.Service, error) {
	return client.CoreV1().Services(namespace).Get(mayaApiServerService, metav1.GetOptions{})
}

// SetupMayaConfig initializes mayaConfig object with mapi server url etc
func (mayaConfig *MayaConfig) SetupMayaConfig(k8sClient K8sClientService) error {

	// setup Namespace
	if mayaConfig.Namespace = os.Getenv("OPENEBS_NAMESPACE"); mayaConfig.Namespace == "" {
		mayaConfig.Namespace = "default"
	}
	glog.V(3).Info("OpenEBS volume plugin namespace ", mayaConfig.Namespace)

	client, err := k8sClient.getK8sClient()
	if err != nil {
		return err
	}

	// setup MapiURI using the Maya API Server Service
	svc, err := k8sClient.getSvcObject(client, mayaConfig.Namespace)
	if err != nil {
		glog.Errorf("Error getting maya-apiserver service object: ", err)
		return err
	}
	glog.V(4).Infof("Maya apiserver spec %v", svc)

	mapiUrl, err := url.Parse("http://" + svc.Spec.ClusterIP + ":" + fmt.Sprintf("%d", svc.Spec.Ports[0].Port))
	if err != nil {
		glog.Errorf("Could not parse maya-apiserver server url: %v", err)
		return err
	}
	mayaConfig.MapiURI = *mapiUrl
	glog.V(3).Infof("Maya Cluster IP: %v", mayaConfig.MapiURI)
	return nil
}

// GetNewMayaConfig creates an object of MayaConfig and calls setup method on it
func (builder MayaConfigBuilder) GetNewMayaConfig(clientWrapper *K8sClientWrapper) (*MayaConfig, error) {
	if clientWrapper == nil {
		clientWrapper = &K8sClientWrapper{ClientService: K8sClient{}}
	}
	config := &MayaConfig{MayaService: &MayaService{}}
	err := config.SetupMayaConfig(clientWrapper.ClientService)
	if err != nil {
		config = nil
	}
	return config, err
}
