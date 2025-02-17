// This file is part of nullboot
// Copyright 2021 Canonical Ltd.
// SPDX-License-Identifier: GPL-3.0-only

// This file does not contain actual tests, but contains mock implementations of EFIVariables

package efibootmgr

import (
	"errors"

	"github.com/canonical/go-efilib"
	efi_linux "github.com/canonical/go-efilib/linux"
)

type NoEFIVariables struct{}

func (NoEFIVariables) ListVariables() ([]efi.VariableDescriptor, error) {
	return nil, efi.ErrVarsUnavailable
}

func (NoEFIVariables) GetVariable(guid efi.GUID, name string) ([]byte, efi.VariableAttributes, error) {
	return nil, 0, efi.ErrVarsUnavailable
}

func (NoEFIVariables) SetVariable(guid efi.GUID, name string, data []byte, attrs efi.VariableAttributes) error {
	return efi.ErrVarsUnavailable
}

func (NoEFIVariables) NewFileDevicePath(filepath string, mode efi_linux.FileDevicePathMode) (efi.DevicePath, error) {
	return nil, errors.New("Cannot access")
}

type mockEFIVariable struct {
	data  []byte
	attrs efi.VariableAttributes
}

type MockEFIVariables struct {
	store map[efi.VariableDescriptor]mockEFIVariable
}

func (m MockEFIVariables) ListVariables() (out []efi.VariableDescriptor, err error) {
	for k := range m.store {
		out = append(out, k)
	}
	return out, nil
}

func (m MockEFIVariables) GetVariable(guid efi.GUID, name string) (data []byte, attrs efi.VariableAttributes, err error) {
	out, ok := m.store[efi.VariableDescriptor{Name: name, GUID: guid}]
	if !ok {
		return nil, 0, efi.ErrVarNotExist
	}
	return out.data, out.attrs, nil
}

func (m *MockEFIVariables) SetVariable(guid efi.GUID, name string, data []byte, attrs efi.VariableAttributes) error {
	if m.store == nil {
		m.store = make(map[efi.VariableDescriptor]mockEFIVariable)
	}
	if len(data) == 0 {
		delete(m.store, efi.VariableDescriptor{Name: name, GUID: guid})
	} else {
		m.store[efi.VariableDescriptor{Name: name, GUID: guid}] = mockEFIVariable{data, attrs}
	}
	return nil
}

func (m MockEFIVariables) NewFileDevicePath(filepath string, mode efi_linux.FileDevicePathMode) (efi.DevicePath, error) {
	file, err := appFs.Open(filepath)
	if err != nil {
		return nil, err
	}
	file.Close()

	return efi.DevicePath{
		&efi.ACPIDevicePathNode{HID: 0x0a0341d0},
		&efi.PCIDevicePathNode{Device: 0x14, Function: 0},
		&efi.USBDevicePathNode{ParentPortNumber: 0xb, InterfaceNumber: 0x1}}, nil
}

var (
	UsbrBootCdromOpt = &efi.LoadOption{
		Attributes:  efi.LoadOptionActive | efi.LoadOptionHidden,
		Description: "USBR BOOT CDROM",
		FilePath: efi.DevicePath{
			&efi.ACPIDevicePathNode{HID: 0x0a0341d0},
			&efi.PCIDevicePathNode{Device: 0x14, Function: 0},
			&efi.USBDevicePathNode{ParentPortNumber: 0xb, InterfaceNumber: 0x1}},
		OptionalData: []byte{}}
	UsbrBootCdromOptBytes []byte
)

func init() {
	var err error
	UsbrBootCdromOptBytes, err = UsbrBootCdromOpt.Bytes()
	if err != nil {
		panic(err)
	}
}
