// Code generated by MockGen. DO NOT EDIT.
// Source: ./limit.go
//
// Generated by this command:
//
//	mockgen -source=./limit.go -destination=./limit_mock.go -package=rcmgr
//

// Package rcmgr is a generated GoMock package.
package rcmgr

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockLimit is a mock of Limit interface.
type MockLimit struct {
	ctrl     *gomock.Controller
	recorder *MockLimitMockRecorder
}

// MockLimitMockRecorder is the mock recorder for MockLimit.
type MockLimitMockRecorder struct {
	mock *MockLimit
}

// NewMockLimit creates a new mock instance.
func NewMockLimit(ctrl *gomock.Controller) *MockLimit {
	mock := &MockLimit{ctrl: ctrl}
	mock.recorder = &MockLimitMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockLimit) EXPECT() *MockLimitMockRecorder {
	return m.recorder
}

// Add mocks base method.
func (m *MockLimit) Add(arg0 Limit) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Add", arg0)
}

// Add indicates an expected call of Add.
func (mr *MockLimitMockRecorder) Add(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Add", reflect.TypeOf((*MockLimit)(nil).Add), arg0)
}

// Equal mocks base method.
func (m *MockLimit) Equal(arg0 Limit) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Equal", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// Equal indicates an expected call of Equal.
func (mr *MockLimitMockRecorder) Equal(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Equal", reflect.TypeOf((*MockLimit)(nil).Equal), arg0)
}

// GetConnLimit mocks base method.
func (m *MockLimit) GetConnLimit(arg0 Direction) int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetConnLimit", arg0)
	ret0, _ := ret[0].(int)
	return ret0
}

// GetConnLimit indicates an expected call of GetConnLimit.
func (mr *MockLimitMockRecorder) GetConnLimit(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetConnLimit", reflect.TypeOf((*MockLimit)(nil).GetConnLimit), arg0)
}

// GetConnTotalLimit mocks base method.
func (m *MockLimit) GetConnTotalLimit() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetConnTotalLimit")
	ret0, _ := ret[0].(int)
	return ret0
}

// GetConnTotalLimit indicates an expected call of GetConnTotalLimit.
func (mr *MockLimitMockRecorder) GetConnTotalLimit() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetConnTotalLimit", reflect.TypeOf((*MockLimit)(nil).GetConnTotalLimit))
}

// GetFDLimit mocks base method.
func (m *MockLimit) GetFDLimit() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFDLimit")
	ret0, _ := ret[0].(int)
	return ret0
}

// GetFDLimit indicates an expected call of GetFDLimit.
func (mr *MockLimitMockRecorder) GetFDLimit() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFDLimit", reflect.TypeOf((*MockLimit)(nil).GetFDLimit))
}

// GetMemoryLimit mocks base method.
func (m *MockLimit) GetMemoryLimit() int64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMemoryLimit")
	ret0, _ := ret[0].(int64)
	return ret0
}

// GetMemoryLimit indicates an expected call of GetMemoryLimit.
func (mr *MockLimitMockRecorder) GetMemoryLimit() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMemoryLimit", reflect.TypeOf((*MockLimit)(nil).GetMemoryLimit))
}

// GetTaskLimit mocks base method.
func (m *MockLimit) GetTaskLimit(arg0 ReserveTaskPriority) int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTaskLimit", arg0)
	ret0, _ := ret[0].(int)
	return ret0
}

// GetTaskLimit indicates an expected call of GetTaskLimit.
func (mr *MockLimitMockRecorder) GetTaskLimit(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTaskLimit", reflect.TypeOf((*MockLimit)(nil).GetTaskLimit), arg0)
}

// GetTaskTotalLimit mocks base method.
func (m *MockLimit) GetTaskTotalLimit() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTaskTotalLimit")
	ret0, _ := ret[0].(int)
	return ret0
}

// GetTaskTotalLimit indicates an expected call of GetTaskTotalLimit.
func (mr *MockLimitMockRecorder) GetTaskTotalLimit() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTaskTotalLimit", reflect.TypeOf((*MockLimit)(nil).GetTaskTotalLimit))
}

// NotLess mocks base method.
func (m *MockLimit) NotLess(arg0 Limit) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "NotLess", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// NotLess indicates an expected call of NotLess.
func (mr *MockLimitMockRecorder) NotLess(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "NotLess", reflect.TypeOf((*MockLimit)(nil).NotLess), arg0)
}

// ScopeStat mocks base method.
func (m *MockLimit) ScopeStat() *ScopeStat {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ScopeStat")
	ret0, _ := ret[0].(*ScopeStat)
	return ret0
}

// ScopeStat indicates an expected call of ScopeStat.
func (mr *MockLimitMockRecorder) ScopeStat() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ScopeStat", reflect.TypeOf((*MockLimit)(nil).ScopeStat))
}

// String mocks base method.
func (m *MockLimit) String() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "String")
	ret0, _ := ret[0].(string)
	return ret0
}

// String indicates an expected call of String.
func (mr *MockLimitMockRecorder) String() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "String", reflect.TypeOf((*MockLimit)(nil).String))
}

// Sub mocks base method.
func (m *MockLimit) Sub(arg0 Limit) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Sub", arg0)
	ret0, _ := ret[0].(bool)
	return ret0
}

// Sub indicates an expected call of Sub.
func (mr *MockLimitMockRecorder) Sub(arg0 any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Sub", reflect.TypeOf((*MockLimit)(nil).Sub), arg0)
}

// MockLimiter is a mock of Limiter interface.
type MockLimiter struct {
	ctrl     *gomock.Controller
	recorder *MockLimiterMockRecorder
}

// MockLimiterMockRecorder is the mock recorder for MockLimiter.
type MockLimiterMockRecorder struct {
	mock *MockLimiter
}

// NewMockLimiter creates a new mock instance.
func NewMockLimiter(ctrl *gomock.Controller) *MockLimiter {
	mock := &MockLimiter{ctrl: ctrl}
	mock.recorder = &MockLimiterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockLimiter) EXPECT() *MockLimiterMockRecorder {
	return m.recorder
}

// GetServiceLimits mocks base method.
func (m *MockLimiter) GetServiceLimits(svc string) Limit {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetServiceLimits", svc)
	ret0, _ := ret[0].(Limit)
	return ret0
}

// GetServiceLimits indicates an expected call of GetServiceLimits.
func (mr *MockLimiterMockRecorder) GetServiceLimits(svc any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetServiceLimits", reflect.TypeOf((*MockLimiter)(nil).GetServiceLimits), svc)
}

// GetSystemLimits mocks base method.
func (m *MockLimiter) GetSystemLimits() Limit {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSystemLimits")
	ret0, _ := ret[0].(Limit)
	return ret0
}

// GetSystemLimits indicates an expected call of GetSystemLimits.
func (mr *MockLimiterMockRecorder) GetSystemLimits() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSystemLimits", reflect.TypeOf((*MockLimiter)(nil).GetSystemLimits))
}

// GetTransientLimits mocks base method.
func (m *MockLimiter) GetTransientLimits() Limit {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTransientLimits")
	ret0, _ := ret[0].(Limit)
	return ret0
}

// GetTransientLimits indicates an expected call of GetTransientLimits.
func (mr *MockLimiterMockRecorder) GetTransientLimits() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTransientLimits", reflect.TypeOf((*MockLimiter)(nil).GetTransientLimits))
}

// String mocks base method.
func (m *MockLimiter) String() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "String")
	ret0, _ := ret[0].(string)
	return ret0
}

// String indicates an expected call of String.
func (mr *MockLimiterMockRecorder) String() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "String", reflect.TypeOf((*MockLimiter)(nil).String))
}
