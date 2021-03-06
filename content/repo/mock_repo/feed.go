// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/urandom/readeef/content/repo (interfaces: Feed)

// Package mock_repo is a generated GoMock package.
package mock_repo

import (
	gomock "github.com/golang/mock/gomock"
	content "github.com/urandom/readeef/content"
	reflect "reflect"
)

// MockFeed is a mock of Feed interface
type MockFeed struct {
	ctrl     *gomock.Controller
	recorder *MockFeedMockRecorder
}

// MockFeedMockRecorder is the mock recorder for MockFeed
type MockFeedMockRecorder struct {
	mock *MockFeed
}

// NewMockFeed creates a new mock instance
func NewMockFeed(ctrl *gomock.Controller) *MockFeed {
	mock := &MockFeed{ctrl: ctrl}
	mock.recorder = &MockFeedMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockFeed) EXPECT() *MockFeedMockRecorder {
	return m.recorder
}

// All mocks base method
func (m *MockFeed) All() ([]content.Feed, error) {
	ret := m.ctrl.Call(m, "All")
	ret0, _ := ret[0].([]content.Feed)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// All indicates an expected call of All
func (mr *MockFeedMockRecorder) All() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "All", reflect.TypeOf((*MockFeed)(nil).All))
}

// AttachTo mocks base method
func (m *MockFeed) AttachTo(arg0 content.Feed, arg1 content.User) error {
	ret := m.ctrl.Call(m, "AttachTo", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// AttachTo indicates an expected call of AttachTo
func (mr *MockFeedMockRecorder) AttachTo(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AttachTo", reflect.TypeOf((*MockFeed)(nil).AttachTo), arg0, arg1)
}

// Delete mocks base method
func (m *MockFeed) Delete(arg0 content.Feed) error {
	ret := m.ctrl.Call(m, "Delete", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete
func (mr *MockFeedMockRecorder) Delete(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockFeed)(nil).Delete), arg0)
}

// DetachFrom mocks base method
func (m *MockFeed) DetachFrom(arg0 content.Feed, arg1 content.User) error {
	ret := m.ctrl.Call(m, "DetachFrom", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DetachFrom indicates an expected call of DetachFrom
func (mr *MockFeedMockRecorder) DetachFrom(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DetachFrom", reflect.TypeOf((*MockFeed)(nil).DetachFrom), arg0, arg1)
}

// FindByLink mocks base method
func (m *MockFeed) FindByLink(arg0 string) (content.Feed, error) {
	ret := m.ctrl.Call(m, "FindByLink", arg0)
	ret0, _ := ret[0].(content.Feed)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindByLink indicates an expected call of FindByLink
func (mr *MockFeedMockRecorder) FindByLink(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindByLink", reflect.TypeOf((*MockFeed)(nil).FindByLink), arg0)
}

// ForTag mocks base method
func (m *MockFeed) ForTag(arg0 content.Tag, arg1 content.User) ([]content.Feed, error) {
	ret := m.ctrl.Call(m, "ForTag", arg0, arg1)
	ret0, _ := ret[0].([]content.Feed)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ForTag indicates an expected call of ForTag
func (mr *MockFeedMockRecorder) ForTag(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ForTag", reflect.TypeOf((*MockFeed)(nil).ForTag), arg0, arg1)
}

// ForUser mocks base method
func (m *MockFeed) ForUser(arg0 content.User) ([]content.Feed, error) {
	ret := m.ctrl.Call(m, "ForUser", arg0)
	ret0, _ := ret[0].([]content.Feed)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ForUser indicates an expected call of ForUser
func (mr *MockFeedMockRecorder) ForUser(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ForUser", reflect.TypeOf((*MockFeed)(nil).ForUser), arg0)
}

// Get mocks base method
func (m *MockFeed) Get(arg0 content.FeedID, arg1 content.User) (content.Feed, error) {
	ret := m.ctrl.Call(m, "Get", arg0, arg1)
	ret0, _ := ret[0].(content.Feed)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get
func (mr *MockFeedMockRecorder) Get(arg0, arg1 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockFeed)(nil).Get), arg0, arg1)
}

// IDs mocks base method
func (m *MockFeed) IDs() ([]content.FeedID, error) {
	ret := m.ctrl.Call(m, "IDs")
	ret0, _ := ret[0].([]content.FeedID)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// IDs indicates an expected call of IDs
func (mr *MockFeedMockRecorder) IDs() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IDs", reflect.TypeOf((*MockFeed)(nil).IDs))
}

// SetUserTags mocks base method
func (m *MockFeed) SetUserTags(arg0 content.Feed, arg1 content.User, arg2 []*content.Tag) error {
	ret := m.ctrl.Call(m, "SetUserTags", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetUserTags indicates an expected call of SetUserTags
func (mr *MockFeedMockRecorder) SetUserTags(arg0, arg1, arg2 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetUserTags", reflect.TypeOf((*MockFeed)(nil).SetUserTags), arg0, arg1, arg2)
}

// Unsubscribed mocks base method
func (m *MockFeed) Unsubscribed() ([]content.Feed, error) {
	ret := m.ctrl.Call(m, "Unsubscribed")
	ret0, _ := ret[0].([]content.Feed)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Unsubscribed indicates an expected call of Unsubscribed
func (mr *MockFeedMockRecorder) Unsubscribed() *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Unsubscribed", reflect.TypeOf((*MockFeed)(nil).Unsubscribed))
}

// Update mocks base method
func (m *MockFeed) Update(arg0 *content.Feed) ([]content.Article, error) {
	ret := m.ctrl.Call(m, "Update", arg0)
	ret0, _ := ret[0].([]content.Article)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Update indicates an expected call of Update
func (mr *MockFeedMockRecorder) Update(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockFeed)(nil).Update), arg0)
}

// Users mocks base method
func (m *MockFeed) Users(arg0 content.Feed) ([]content.User, error) {
	ret := m.ctrl.Call(m, "Users", arg0)
	ret0, _ := ret[0].([]content.User)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Users indicates an expected call of Users
func (mr *MockFeedMockRecorder) Users(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Users", reflect.TypeOf((*MockFeed)(nil).Users), arg0)
}
