package client

import (
	"k8s.io/client-go/rest"
	clientset "qiniu.io/rio-csi/generated/internalclientset"
)

type CsiClient interface {
	Storage() *clientset.Clientset
	Config() *rest.Config
}

type csiclient struct {
	config  *rest.Config
	storage *clientset.Clientset
}

func (c csiclient) Storage() *clientset.Clientset {
	return c.storage
}

func (c csiclient) Config() *rest.Config {
	return c.config
}

var _ CsiClient = csiclient{}
