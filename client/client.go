package client

import (
	"fmt"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	clientset "qiniu.io/rio-csi/generated/internalclientset"
)

type KubeClient struct {
	ClientSet *clientset.Clientset
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

	clientSet, err := clientset.NewForConfig(config)
	if err != nil {
		return
	}

	c = &KubeClient{
		ClientSet: clientSet,
	}
	return
}
