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

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"time"

	v1alpha1 "github.com/kyma-project/kyma/components/event-sources/apis/sources/v1alpha1"
	scheme "github.com/kyma-project/kyma/components/event-sources/client/generated/clientset/internalclientset/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// MQTTSourcesGetter has a method to return a MQTTSourceInterface.
// A group's client should implement this interface.
type MQTTSourcesGetter interface {
	MQTTSources(namespace string) MQTTSourceInterface
}

// MQTTSourceInterface has methods to work with MQTTSource resources.
type MQTTSourceInterface interface {
	Create(*v1alpha1.MQTTSource) (*v1alpha1.MQTTSource, error)
	Update(*v1alpha1.MQTTSource) (*v1alpha1.MQTTSource, error)
	UpdateStatus(*v1alpha1.MQTTSource) (*v1alpha1.MQTTSource, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.MQTTSource, error)
	List(opts v1.ListOptions) (*v1alpha1.MQTTSourceList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.MQTTSource, err error)
	MQTTSourceExpansion
}

// mQTTSources implements MQTTSourceInterface
type mQTTSources struct {
	client rest.Interface
	ns     string
}

// newMQTTSources returns a MQTTSources
func newMQTTSources(c *SourcesV1alpha1Client, namespace string) *mQTTSources {
	return &mQTTSources{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the mQTTSource, and returns the corresponding mQTTSource object, and an error if there is any.
func (c *mQTTSources) Get(name string, options v1.GetOptions) (result *v1alpha1.MQTTSource, err error) {
	result = &v1alpha1.MQTTSource{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("mqttsources").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of MQTTSources that match those selectors.
func (c *mQTTSources) List(opts v1.ListOptions) (result *v1alpha1.MQTTSourceList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.MQTTSourceList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("mqttsources").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested mQTTSources.
func (c *mQTTSources) Watch(opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("mqttsources").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch()
}

// Create takes the representation of a mQTTSource and creates it.  Returns the server's representation of the mQTTSource, and an error, if there is any.
func (c *mQTTSources) Create(mQTTSource *v1alpha1.MQTTSource) (result *v1alpha1.MQTTSource, err error) {
	result = &v1alpha1.MQTTSource{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("mqttsources").
		Body(mQTTSource).
		Do().
		Into(result)
	return
}

// Update takes the representation of a mQTTSource and updates it. Returns the server's representation of the mQTTSource, and an error, if there is any.
func (c *mQTTSources) Update(mQTTSource *v1alpha1.MQTTSource) (result *v1alpha1.MQTTSource, err error) {
	result = &v1alpha1.MQTTSource{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("mqttsources").
		Name(mQTTSource.Name).
		Body(mQTTSource).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *mQTTSources) UpdateStatus(mQTTSource *v1alpha1.MQTTSource) (result *v1alpha1.MQTTSource, err error) {
	result = &v1alpha1.MQTTSource{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("mqttsources").
		Name(mQTTSource.Name).
		SubResource("status").
		Body(mQTTSource).
		Do().
		Into(result)
	return
}

// Delete takes name of the mQTTSource and deletes it. Returns an error if one occurs.
func (c *mQTTSources) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("mqttsources").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *mQTTSources) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	var timeout time.Duration
	if listOptions.TimeoutSeconds != nil {
		timeout = time.Duration(*listOptions.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("mqttsources").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Timeout(timeout).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched mQTTSource.
func (c *mQTTSources) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.MQTTSource, err error) {
	result = &v1alpha1.MQTTSource{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("mqttsources").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
