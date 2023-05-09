// Code generated by MockGen. DO NOT EDIT.
// Source: filestore/file_store.go

// Package mocks is a generated GoMock package.
package mocks

import (
	io "io"
	reflect "reflect"
	time "time"

	filestore "github.com/ristomcgehee/lyncser/filestore"
	gomock "github.com/golang/mock/gomock"
)

// MockFileStore is a mock of FileStore interface.
type MockFileStore struct {
	ctrl     *gomock.Controller
	recorder *MockFileStoreMockRecorder
}

// MockFileStoreMockRecorder is the mock recorder for MockFileStore.
type MockFileStoreMockRecorder struct {
	mock *MockFileStore
}

// NewMockFileStore creates a new mock instance.
func NewMockFileStore(ctrl *gomock.Controller) *MockFileStore {
	mock := &MockFileStore{ctrl: ctrl}
	mock.recorder = &MockFileStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockFileStore) EXPECT() *MockFileStoreMockRecorder {
	return m.recorder
}

// DeleteAllFiles mocks base method.
func (m *MockFileStore) DeleteAllFiles() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteAllFiles")
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteAllFiles indicates an expected call of DeleteAllFiles.
func (mr *MockFileStoreMockRecorder) DeleteAllFiles() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteAllFiles", reflect.TypeOf((*MockFileStore)(nil).DeleteAllFiles))
}

// DeleteFile mocks base method.
func (m *MockFileStore) DeleteFile(path string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteFile", path)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteFile indicates an expected call of DeleteFile.
func (mr *MockFileStoreMockRecorder) DeleteFile(path interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteFile", reflect.TypeOf((*MockFileStore)(nil).DeleteFile), path)
}

// FileExists mocks base method.
func (m *MockFileStore) FileExists(path string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FileExists", path)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FileExists indicates an expected call of FileExists.
func (mr *MockFileStoreMockRecorder) FileExists(path interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FileExists", reflect.TypeOf((*MockFileStore)(nil).FileExists), path)
}

// GetFileContents mocks base method.
func (m *MockFileStore) GetFileContents(path string) (io.ReadCloser, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFileContents", path)
	ret0, _ := ret[0].(io.ReadCloser)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetFileContents indicates an expected call of GetFileContents.
func (mr *MockFileStoreMockRecorder) GetFileContents(path interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFileContents", reflect.TypeOf((*MockFileStore)(nil).GetFileContents), path)
}

// GetFiles mocks base method.
func (m *MockFileStore) GetFiles() ([]*filestore.StoredFile, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFiles")
	ret0, _ := ret[0].([]*filestore.StoredFile)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetFiles indicates an expected call of GetFiles.
func (mr *MockFileStoreMockRecorder) GetFiles() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFiles", reflect.TypeOf((*MockFileStore)(nil).GetFiles))
}

// GetModifiedTime mocks base method.
func (m *MockFileStore) GetModifiedTime(path string) (time.Time, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetModifiedTime", path)
	ret0, _ := ret[0].(time.Time)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetModifiedTime indicates an expected call of GetModifiedTime.
func (mr *MockFileStoreMockRecorder) GetModifiedTime(path interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetModifiedTime", reflect.TypeOf((*MockFileStore)(nil).GetModifiedTime), path)
}

// WriteFileContents mocks base method.
func (m *MockFileStore) WriteFileContents(path string, contentReader io.Reader) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "WriteFileContents", path, contentReader)
	ret0, _ := ret[0].(error)
	return ret0
}

// WriteFileContents indicates an expected call of WriteFileContents.
func (mr *MockFileStoreMockRecorder) WriteFileContents(path, contentReader interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WriteFileContents", reflect.TypeOf((*MockFileStore)(nil).WriteFileContents), path, contentReader)
}
