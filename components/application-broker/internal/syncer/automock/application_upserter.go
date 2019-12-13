// Code generated by mockery v1.0.0. DO NOT EDIT.

package automock

import (
	internal "github.com/kyma-project/kyma/components/application-broker/internal"
	mock "github.com/stretchr/testify/mock"
)

// ApplicationUpserter is an autogenerated mock type for the ApplicationUpserter type
type ApplicationUpserter struct {
	mock.Mock
}

// Upsert provides a mock function with given fields: app
func (_m *ApplicationUpserter) Upsert(app *internal.Application) (bool, error) {
	ret := _m.Called(app)

	var r0 bool
	if rf, ok := ret.Get(0).(func(*internal.Application) bool); ok {
		r0 = rf(app)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*internal.Application) error); ok {
		r1 = rf(app)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
