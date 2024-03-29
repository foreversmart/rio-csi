/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	v1 "qiniu.io/rio-csi/api/rio/v1"
)

// RioNodeLister helps list RioNodes.
// All objects returned here must be treated as read-only.
type RioNodeLister interface {
	// List lists all RioNodes in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.RioNode, err error)
	// RioNodes returns an object that can list and get RioNodes.
	RioNodes(namespace string) RioNodeNamespaceLister
	RioNodeListerExpansion
}

// rioNodeLister implements the RioNodeLister interface.
type rioNodeLister struct {
	indexer cache.Indexer
}

// NewRioNodeLister returns a new RioNodeLister.
func NewRioNodeLister(indexer cache.Indexer) RioNodeLister {
	return &rioNodeLister{indexer: indexer}
}

// List lists all RioNodes in the indexer.
func (s *rioNodeLister) List(selector labels.Selector) (ret []*v1.RioNode, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.RioNode))
	})
	return ret, err
}

// RioNodes returns an object that can list and get RioNodes.
func (s *rioNodeLister) RioNodes(namespace string) RioNodeNamespaceLister {
	return rioNodeNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// RioNodeNamespaceLister helps list and get RioNodes.
// All objects returned here must be treated as read-only.
type RioNodeNamespaceLister interface {
	// List lists all RioNodes in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.RioNode, err error)
	// Get retrieves the RioNode from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.RioNode, error)
	RioNodeNamespaceListerExpansion
}

// rioNodeNamespaceLister implements the RioNodeNamespaceLister
// interface.
type rioNodeNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all RioNodes in the indexer for a given namespace.
func (s rioNodeNamespaceLister) List(selector labels.Selector) (ret []*v1.RioNode, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.RioNode))
	})
	return ret, err
}

// Get retrieves the RioNode from the indexer for a given namespace and name.
func (s rioNodeNamespaceLister) Get(name string) (*v1.RioNode, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("rionode"), name)
	}
	return obj.(*v1.RioNode), nil
}
