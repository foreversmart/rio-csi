package informer

import (
	"qiniu.io/rio-csi/client"
	v1 "qiniu.io/rio-csi/generated/informer/externalversions/rio/v1"
	"sync"
	"time"

	"qiniu.io/rio-csi/generated/informer/externalversions"
)

type Storage interface {
	Volume() v1.VolumeInformer
	RioNode() v1.RioNodeInformer
	Snapshot() v1.SnapshotInformer
}

type InformerFactory interface {
	Storage

	Start(stopCh <-chan struct{})
	WaitForCacheSync(stopCh <-chan struct{})
}

type informerFactory struct {
	lock          sync.Mutex
	defaultResync time.Duration

	clientSet client.CsiClient

	csiInformers externalversions.SharedInformerFactory
}

func New(client client.CsiClient, stopCh chan struct{}) InformerFactory {
	informer := &informerFactory{
		clientSet:    client,
		csiInformers: externalversions.NewSharedInformerFactory(client.Storage(), 0),
	}
	return initCsiCache(informer, stopCh)
}

// Start can be called from multiple controllers in different go routines safely.
// Only informers that have not started are triggered by this function.
// Multiple calls to this function are idempotent.
func (f *informerFactory) Start(stopCh <-chan struct{}) {
	f.csiInformers.Start(stopCh)
}

func (f *informerFactory) WaitForCacheSync(stopCh <-chan struct{}) {
	f.csiInformers.WaitForCacheSync(stopCh)
}

func (f *informerFactory) Volume() v1.VolumeInformer {
	return f.csiInformers.Rio().V1().Volumes()
}
func (f *informerFactory) Snapshot() v1.SnapshotInformer {
	return f.csiInformers.Rio().V1().Snapshots()
}
func (f *informerFactory) RioNode() v1.RioNodeInformer {
	return f.csiInformers.Rio().V1().RioNodes()
}
func initCsiCache(cache InformerFactory, stopCh chan struct{}) InformerFactory {

	//add register
	_ = cache.Volume().Informer()
	_ = cache.Snapshot().Informer()
	_ = cache.RioNode().Informer()

	cache.Start(stopCh)
	cache.WaitForCacheSync(stopCh)
	return cache
}
