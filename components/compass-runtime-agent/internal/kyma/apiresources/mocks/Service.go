// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import (
	mock "github.com/stretchr/testify/mock"
	apperrors "kyma-project.io/compass-runtime-agent/internal/apperrors"
	clusterassetgroup "kyma-project.io/compass-runtime-agent/internal/kyma/apiresources/rafter/clusterassetgroup"

	model "kyma-project.io/compass-runtime-agent/internal/kyma/apiresources/secrets/model"

	types "k8s.io/apimachinery/pkg/types"
)

// Service is an autogenerated mock type for the Service type
type Service struct {
	mock.Mock
}

// CreateApiResources provides a mock function with given fields: applicationName, applicationUID, serviceID, credentials, spec, specFormat, apiType
func (_m *Service) CreateApiResources(applicationName string, applicationUID types.UID, serviceID string, credentials *model.CredentialsWithCSRF, spec []byte, specFormat clusterassetgroup.SpecFormat, apiType clusterassetgroup.ApiType) apperrors.AppError {
	ret := _m.Called(applicationName, applicationUID, serviceID, credentials, spec, specFormat, apiType)

	var r0 apperrors.AppError
	if rf, ok := ret.Get(0).(func(string, types.UID, string, *model.CredentialsWithCSRF, []byte, clusterassetgroup.SpecFormat, clusterassetgroup.ApiType) apperrors.AppError); ok {
		r0 = rf(applicationName, applicationUID, serviceID, credentials, spec, specFormat, apiType)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(apperrors.AppError)
		}
	}

	return r0
}

// CreateEventApiResources provides a mock function with given fields: applicationName, serviceID, spec, specFormat, apiType
func (_m *Service) CreateEventApiResources(applicationName string, serviceID string, spec []byte, specFormat clusterassetgroup.SpecFormat, apiType clusterassetgroup.ApiType) apperrors.AppError {
	ret := _m.Called(applicationName, serviceID, spec, specFormat, apiType)

	var r0 apperrors.AppError
	if rf, ok := ret.Get(0).(func(string, string, []byte, clusterassetgroup.SpecFormat, clusterassetgroup.ApiType) apperrors.AppError); ok {
		r0 = rf(applicationName, serviceID, spec, specFormat, apiType)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(apperrors.AppError)
		}
	}

	return r0
}

// DeleteApiResources provides a mock function with given fields: applicationName, serviceID, secretName
func (_m *Service) DeleteApiResources(applicationName string, serviceID string, secretName string) apperrors.AppError {
	ret := _m.Called(applicationName, serviceID, secretName)

	var r0 apperrors.AppError
	if rf, ok := ret.Get(0).(func(string, string, string) apperrors.AppError); ok {
		r0 = rf(applicationName, serviceID, secretName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(apperrors.AppError)
		}
	}

	return r0
}

// UpdateApiResources provides a mock function with given fields: applicationName, applicationUID, serviceID, credentials, spec, specFormat, apiType
func (_m *Service) UpdateApiResources(applicationName string, applicationUID types.UID, serviceID string, credentials *model.CredentialsWithCSRF, spec []byte, specFormat clusterassetgroup.SpecFormat, apiType clusterassetgroup.ApiType) apperrors.AppError {
	ret := _m.Called(applicationName, applicationUID, serviceID, credentials, spec, specFormat, apiType)

	var r0 apperrors.AppError
	if rf, ok := ret.Get(0).(func(string, types.UID, string, *model.CredentialsWithCSRF, []byte, clusterassetgroup.SpecFormat, clusterassetgroup.ApiType) apperrors.AppError); ok {
		r0 = rf(applicationName, applicationUID, serviceID, credentials, spec, specFormat, apiType)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(apperrors.AppError)
		}
	}

	return r0
}

// UpdateEventApiResources provides a mock function with given fields: applicationName, serviceID, spec, specFormat, apiType
func (_m *Service) UpdateEventApiResources(applicationName string, serviceID string, spec []byte, specFormat clusterassetgroup.SpecFormat, apiType clusterassetgroup.ApiType) apperrors.AppError {
	ret := _m.Called(applicationName, serviceID, spec, specFormat, apiType)

	var r0 apperrors.AppError
	if rf, ok := ret.Get(0).(func(string, string, []byte, clusterassetgroup.SpecFormat, clusterassetgroup.ApiType) apperrors.AppError); ok {
		r0 = rf(applicationName, serviceID, spec, specFormat, apiType)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(apperrors.AppError)
		}
	}

	return r0
}
