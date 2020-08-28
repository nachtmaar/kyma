// Code generated by mockery v2.1.0. DO NOT EDIT.

package mocks

import (
	v1alpha1 "github.com/kyma-project/kyma/components/application-operator/pkg/apis/applicationconnector/v1alpha1"
	model "github.com/kyma-project/kyma/components/compass-runtime-agent/internal/kyma/model"
	mock "github.com/stretchr/testify/mock"
)

// Converter is an autogenerated mock type for the Converter type
type Converter struct {
	mock.Mock
}

// Do provides a mock function with given fields: application
func (_m *Converter) Do(application model.Application) v1alpha1.Application {
	ret := _m.Called(application)

	var r0 v1alpha1.Application
	if rf, ok := ret.Get(0).(func(model.Application) v1alpha1.Application); ok {
		r0 = rf(application)
	} else {
		r0 = ret.Get(0).(v1alpha1.Application)
	}

	return r0
}
