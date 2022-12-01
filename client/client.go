package client

import (
	"fmt"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	clientset "qiniu.io/rio-csi/generated/internalclientset"
)

type KubeClient struct {
	InternalClientSet *clientset.Clientset
	ClientSet         *kubernetes.Clientset
	DynamicClient     dynamic.Interface
}

var (
	DefaultClient *KubeClient
)

func init() {
	c, err := NewDefault("", "")
	if err != nil {
		panic(err)
	}

	DefaultClient = c
}

// NewDefault TODO master url
func NewDefault(masterUrl, kubeConfigPath string) (c *KubeClient, err error) {
	config, err := clientcmd.BuildConfigFromFlags(masterUrl, kubeConfigPath)
	if err != nil {
		fmt.Printf("The kubeconfig cannot be loaded: %v\n", err)
		os.Exit(1)
	}

	internalClientSet, err := clientset.NewForConfig(config)
	if err != nil {
		return
	}

	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return
	}

	dc, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	c = &KubeClient{
		InternalClientSet: internalClientSet,
		ClientSet:         cs,
		DynamicClient:     dc,
	}
	return
}
