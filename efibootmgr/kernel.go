// This file is part of bootmgrless
// Copyright 2021 Canonical Ltd.
// SPDX-License-Identifier: GPL-3.0-only

package efibootmgr

import (
	"fmt"
	"log"
	"path"
	"sort"
	"strings"

	"github.com/knqyf263/go-deb-version"
)

// KernelManager manages kernels in an SP vendor directory.
//
// It will update or install shim, copy in any new kernels,
// remove old kernels, and configure boot in shim and BDS.
type KernelManager struct {
	sourceDir     string      // sourceDir is the location to copy kernels from
	targetDir     string      // targetDir is a vendor directory on the ESP
	sourceKernels []string    // kernels in sourceDir
	targetKernels []string    // kernels in targetDir
	bootEntries   []BootEntry // boot entries filled by InstallKernels
	kernelOptions string      // options to pass to kernel
}

// NewKernelManager returns a new kernel manager managing kernels in the host system
func NewKernelManager() (*KernelManager, error) {
	var km KernelManager
	var err error

	// FIXME: Read dirs and options from a configuration option
	km.sourceDir = "/usr/lib/linux"
	km.targetDir = "/boot/efi/EFI/ubuntu"
	km.kernelOptions = "root=magic"

	km.sourceKernels, err = km.readKernels(km.sourceDir)
	if err != nil {
		return nil, err
	}
	km.targetKernels, err = km.readKernels(km.targetDir)
	if err != nil {
		return nil, err
	}
	return &km, nil
}

// readKernels returns a list of all kernels in the
func (km *KernelManager) readKernels(dir string) ([]string, error) {
	var kernels []string
	entries, err := appFs.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("Could not determine kernels: %w", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "kernel.efi-") {
			kernels = append(kernels, e.Name())
		}
	}
	// Sort descending
	sort.Slice(kernels, func(i, j int) bool {
		a, e := version.NewVersion(kernels[i][len("kernel.efi-"):])
		if e != nil {
			err = fmt.Errorf("Could not parse kernel version of %s: %w", kernels[i], e)
			return false
		}
		b, e := version.NewVersion(kernels[j][len("kernel.efi-"):])
		if e != nil {
			err = fmt.Errorf("Could not parse kernel version of %s: %w", kernels[j], e)
			return false
		}
		return a.GreaterThan(b)
	})
	return kernels, err
}

// getKernelABI returns the kernel ABI part of the kernel filename
func getKernelABI(kernel string) string {
	return kernel[len("kernel.efi-"):]
}

// InstallKernels installs the kernels to the ESP and builds up the boot entries
// to commit using CommitToBootLoader()
func (km *KernelManager) InstallKernels() error {
	km.bootEntries = nil
	for _, sk := range km.sourceKernels {
		updated, err := MaybeUpdateFile(path.Join(km.targetDir, sk),
			path.Join(km.sourceDir, sk))
		if err != nil {
			log.Printf("Could not install kernel %s: %v", sk, err)
			continue
		}
		if updated {
			log.Printf("Installed or updated kernel %s", sk)
		}
		// It is worth pointing out that the argument for shim should start with \
		// which here somehow denotes it is in the same directory rather than the root.
		// FIXME: Extract vendor name out into config file
		skVersion := getKernelABI(sk)
		km.bootEntries = append(km.bootEntries, BootEntry{
			Filename:    "shim" + GetEfiArchitecture() + ".efi",
			Label:       fmt.Sprintf("Ubuntu with kernel %s", skVersion),
			Options:     "\\" + sk + " " + km.kernelOptions,
			Description: fmt.Sprintf("Ubuntu entry for kernel %s", skVersion),
		})
	}

	return nil
}

// IsObsoleteKernel checks whether a kernel is obsolete.
func (km *KernelManager) isObsoleteKernel(k string) bool {
	for _, sk := range km.sourceKernels {
		if sk == k {
			return false
		}
	}
	return true
}

// RemoveObsoleteKernels removes old kernels in the ESP vendor directory
func (km *KernelManager) RemoveObsoleteKernels() error {
	var remaining []string
	for _, tk := range km.targetKernels {
		if !km.isObsoleteKernel(tk) {
			continue
		}
		if err := appFs.Remove(path.Join(km.targetDir, tk)); err != nil {
			log.Printf("Could not remove kernel %s: %v", tk, err)
			remaining = append(remaining, tk)
			continue
		}

		log.Printf("Removed kernel %s", tk)
	}

	km.targetKernels = remaining

	return nil
}

// CommitToBootLoader updates the firmware BDS entries and shim's boot.csv
func (km *KernelManager) CommitToBootLoader() error {
	log.Print("Configuring shim fallback loader")

	// We completely own the shim fallback file, so just write it
	if err := WriteShimFallbackToFile(path.Join(km.targetDir, "BOOT"+strings.ToUpper(GetEfiArchitecture())+".CSV"), km.bootEntries); err != nil {
		log.Printf("Failed to configure shim fallback loader: %v", err)
	}

	log.Print("Configuring UEFI boot device selection")
	// FIXME: Configure BDS
	return nil
}
