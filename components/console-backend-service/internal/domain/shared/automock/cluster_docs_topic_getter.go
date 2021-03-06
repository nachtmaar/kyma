// Code generated by mockery v1.0.0. DO NOT EDIT.
package automock

import mock "github.com/stretchr/testify/mock"

import v1alpha1 "github.com/kyma-project/kyma/components/cms-controller-manager/pkg/apis/cms/v1alpha1"

// ClusterDocsTopicGetter is an autogenerated mock type for the ClusterDocsTopicGetter type
type ClusterDocsTopicGetter struct {
	mock.Mock
}

// Find provides a mock function with given fields: name
func (_m *ClusterDocsTopicGetter) Find(name string) (*v1alpha1.ClusterDocsTopic, error) {
	ret := _m.Called(name)

	var r0 *v1alpha1.ClusterDocsTopic
	if rf, ok := ret.Get(0).(func(string) *v1alpha1.ClusterDocsTopic); ok {
		r0 = rf(name)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*v1alpha1.ClusterDocsTopic)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(name)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
