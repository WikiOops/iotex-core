// Code generated by MockGen. DO NOT EDIT.
// Source: ./state/factory.go

// Package mock_state is a generated GoMock package.
package mock_state

import (
	gomock "github.com/golang/mock/gomock"
	action "github.com/iotexproject/iotex-core/blockchain/action"
	common "github.com/iotexproject/iotex-core/common"
	state "github.com/iotexproject/iotex-core/state"
	big "math/big"
	reflect "reflect"
)

// MockFactory is a mock of Factory interface
type MockFactory struct {
	ctrl     *gomock.Controller
	recorder *MockFactoryMockRecorder
}

// MockFactoryMockRecorder is the mock recorder for MockFactory
type MockFactoryMockRecorder struct {
	mock *MockFactory
}

// NewMockFactory creates a new mock instance
func NewMockFactory(ctrl *gomock.Controller) *MockFactory {
	mock := &MockFactory{ctrl: ctrl}
	mock.recorder = &MockFactoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockFactory) EXPECT() *MockFactoryMockRecorder {
	return m.recorder
}

// CreateState mocks base method
func (m *MockFactory) CreateState(arg0 string, arg1 uint64) (*state.State, error) {
	ret := m.ctrl.Call(m, "CreateState", arg0, arg1)
	ret0, _ := ret[0].(*state.State)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateState indicates an expected call of CreateState
func (mr *MockFactoryMockRecorder) CreateState(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateState", reflect.TypeOf((*MockFactory)(nil).CreateState), arg0, arg1)
}

// Balance mocks base method
func (m *MockFactory) Balance(arg0 string) (*big.Int, error) {
	ret := m.ctrl.Call(m, "Balance", arg0)
	ret0, _ := ret[0].(*big.Int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Balance indicates an expected call of Balance
func (mr *MockFactoryMockRecorder) Balance(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Balance", reflect.TypeOf((*MockFactory)(nil).Balance), arg0)
}

// CommitStateChanges mocks base method
func (m *MockFactory) CommitStateChanges(arg0 uint64, arg1 []*action.Transfer, arg2 []*action.Vote) error {
	ret := m.ctrl.Call(m, "CommitStateChanges", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// CommitStateChanges indicates an expected call of CommitStateChanges
func (mr *MockFactoryMockRecorder) CommitStateChanges(arg0, arg1, arg2 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CommitStateChanges", reflect.TypeOf((*MockFactory)(nil).CommitStateChanges), arg0, arg1, arg2)
}

// Nonce mocks base method
func (m *MockFactory) Nonce(arg0 string) (uint64, error) {
	ret := m.ctrl.Call(m, "Nonce", arg0)
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Nonce indicates an expected call of Nonce
func (mr *MockFactoryMockRecorder) Nonce(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Nonce", reflect.TypeOf((*MockFactory)(nil).Nonce), arg0)
}

// State mocks base method
func (m *MockFactory) State(arg0 string) (*state.State, error) {
	ret := m.ctrl.Call(m, "State", arg0)
	ret0, _ := ret[0].(*state.State)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// State indicates an expected call of State
func (mr *MockFactoryMockRecorder) State(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "State", reflect.TypeOf((*MockFactory)(nil).State), arg0)
}

// RootHash mocks base method
func (m *MockFactory) RootHash() common.Hash32B {
	ret := m.ctrl.Call(m, "RootHash")
	ret0, _ := ret[0].(common.Hash32B)
	return ret0
}

// RootHash indicates an expected call of RootHash
func (mr *MockFactoryMockRecorder) RootHash() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RootHash", reflect.TypeOf((*MockFactory)(nil).RootHash))
}

// Candidates mocks base method
func (m *MockFactory) Candidates() (uint64, []*state.Candidate) {
	ret := m.ctrl.Call(m, "Candidates")
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].([]*state.Candidate)
	return ret0, ret1
}

// Candidates indicates an expected call of Candidates
func (mr *MockFactoryMockRecorder) Candidates() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Candidates", reflect.TypeOf((*MockFactory)(nil).Candidates))
}
