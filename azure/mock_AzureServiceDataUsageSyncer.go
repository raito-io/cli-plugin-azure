// Code generated by mockery v2.33.2. DO NOT EDIT.

package azure

import (
	context "context"

	config "github.com/raito-io/cli/base/util/config"

	data_usage "github.com/raito-io/cli/base/data_usage"

	mock "github.com/stretchr/testify/mock"

	time "time"
)

// MockAzureServiceDataUsageSyncer is an autogenerated mock type for the AzureServiceDataUsageSyncer type
type MockAzureServiceDataUsageSyncer struct {
	mock.Mock
}

type MockAzureServiceDataUsageSyncer_Expecter struct {
	mock *mock.Mock
}

func (_m *MockAzureServiceDataUsageSyncer) EXPECT() *MockAzureServiceDataUsageSyncer_Expecter {
	return &MockAzureServiceDataUsageSyncer_Expecter{mock: &_m.Mock}
}

// SyncDataUsage provides a mock function with given fields: ctx, startDate, configParams, commit
func (_m *MockAzureServiceDataUsageSyncer) SyncDataUsage(ctx context.Context, startDate time.Time, configParams *config.ConfigMap, commit func(data_usage.Statement) error) error {
	ret := _m.Called(ctx, startDate, configParams, commit)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, time.Time, *config.ConfigMap, func(data_usage.Statement) error) error); ok {
		r0 = rf(ctx, startDate, configParams, commit)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockAzureServiceDataUsageSyncer_SyncDataUsage_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SyncDataUsage'
type MockAzureServiceDataUsageSyncer_SyncDataUsage_Call struct {
	*mock.Call
}

// SyncDataUsage is a helper method to define mock.On call
//   - ctx context.Context
//   - startDate time.Time
//   - configParams *config.ConfigMap
//   - commit func(data_usage.Statement) error
func (_e *MockAzureServiceDataUsageSyncer_Expecter) SyncDataUsage(ctx interface{}, startDate interface{}, configParams interface{}, commit interface{}) *MockAzureServiceDataUsageSyncer_SyncDataUsage_Call {
	return &MockAzureServiceDataUsageSyncer_SyncDataUsage_Call{Call: _e.mock.On("SyncDataUsage", ctx, startDate, configParams, commit)}
}

func (_c *MockAzureServiceDataUsageSyncer_SyncDataUsage_Call) Run(run func(ctx context.Context, startDate time.Time, configParams *config.ConfigMap, commit func(data_usage.Statement) error)) *MockAzureServiceDataUsageSyncer_SyncDataUsage_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(time.Time), args[2].(*config.ConfigMap), args[3].(func(data_usage.Statement) error))
	})
	return _c
}

func (_c *MockAzureServiceDataUsageSyncer_SyncDataUsage_Call) Return(_a0 error) *MockAzureServiceDataUsageSyncer_SyncDataUsage_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockAzureServiceDataUsageSyncer_SyncDataUsage_Call) RunAndReturn(run func(context.Context, time.Time, *config.ConfigMap, func(data_usage.Statement) error) error) *MockAzureServiceDataUsageSyncer_SyncDataUsage_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockAzureServiceDataUsageSyncer creates a new instance of MockAzureServiceDataUsageSyncer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockAzureServiceDataUsageSyncer(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockAzureServiceDataUsageSyncer {
	mock := &MockAzureServiceDataUsageSyncer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
