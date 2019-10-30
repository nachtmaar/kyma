/*
Copyright 2019 The Kyma Authors.

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

package v1alpha1

import (
	v1alpha1 "github.com/kyma-project/kyma/components/event-sources/apis/sources/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// MQTTSourceLister helps list MQTTSources.
type MQTTSourceLister interface {
	// List lists all MQTTSources in the indexer.
	List(selector labels.Selector) (ret []*v1alpha1.MQTTSource, err error)
	// MQTTSources returns an object that can list and get MQTTSources.
	MQTTSources(namespace string) MQTTSourceNamespaceLister
	MQTTSourceListerExpansion
}

// mQTTSourceLister implements the MQTTSourceLister interface.
type mQTTSourceLister struct {
	indexer cache.Indexer
}

// NewMQTTSourceLister returns a new MQTTSourceLister.
func NewMQTTSourceLister(indexer cache.Indexer) MQTTSourceLister {
	return &mQTTSourceLister{indexer: indexer}
}

// List lists all MQTTSources in the indexer.
func (s *mQTTSourceLister) List(selector labels.Selector) (ret []*v1alpha1.MQTTSource, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.MQTTSource))
	})
	return ret, err
}

// MQTTSources returns an object that can list and get MQTTSources.
func (s *mQTTSourceLister) MQTTSources(namespace string) MQTTSourceNamespaceLister {
	return mQTTSourceNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// MQTTSourceNamespaceLister helps list and get MQTTSources.
type MQTTSourceNamespaceLister interface {
	// List lists all MQTTSources in the indexer for a given namespace.
	List(selector labels.Selector) (ret []*v1alpha1.MQTTSource, err error)
	// Get retrieves the MQTTSource from the indexer for a given namespace and name.
	Get(name string) (*v1alpha1.MQTTSource, error)
	MQTTSourceNamespaceListerExpansion
}

// mQTTSourceNamespaceLister implements the MQTTSourceNamespaceLister
// interface.
type mQTTSourceNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all MQTTSources in the indexer for a given namespace.
func (s mQTTSourceNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.MQTTSource, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.MQTTSource))
	})
	return ret, err
}

// Get retrieves the MQTTSource from the indexer for a given namespace and name.
func (s mQTTSourceNamespaceLister) Get(name string) (*v1alpha1.MQTTSource, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("mqttsource"), name)
	}
	return obj.(*v1alpha1.MQTTSource), nil
}
