package client

import (
	"fmt"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"os/signal"
	"qiniu.io/rio-csi/generated/informer/externalversions"
	clientset "qiniu.io/rio-csi/generated/internalclientset"
	"syscall"
)

type KubeClient struct {
	InternalClientSet *clientset.Clientset
	ClientSet         *kubernetes.Clientset
	DynamicClient     dynamic.Interface
}

var (
	DefaultClient   *KubeClient
	DefaultInformer externalversions.SharedInformerFactory
	Codecs          serializer.CodecFactory
)

func SetupClusterConfig() {
	c, err := NewDefault("", "")
	if err != nil {
		panic(err)

	}

	DefaultClient = c
	initInformer()
}

func initInformer() {
	DefaultInformer = externalversions.NewSharedInformerFactory(DefaultClient.InternalClientSet, 0)
	stopCh := make(chan struct{})
	c := make(chan os.Signal, 2)
	signal.Notify(c, []os.Signal{os.Interrupt, syscall.SIGTERM}...)
	go func() {
		<-c
		close(stopCh)
	}()

	// add register
	DefaultInformer.Rio().V1().RioNodes().Informer()
	DefaultInformer.Rio().V1().Snapshots().Informer()
	DefaultInformer.Rio().V1().Volumes().Informer()

	DefaultInformer.Start(stopCh)
	DefaultInformer.WaitForCacheSync(stopCh)
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
func GetCsiClientFromRESTConfig(config *rest.Config) (CsiClient, error) {
	config.NegotiatedSerializer = serializer.WithoutConversionCodecFactory{CodecFactory: Codecs}

	cclient, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &csiclient{
		config:  config,
		storage: cclient,
	}, nil

}
