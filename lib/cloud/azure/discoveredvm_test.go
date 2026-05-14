/*
 * Teleport
 * Copyright (C) 2026  Gravitational, Inc.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package azure

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/stretchr/testify/require"
)

func TestDiscoveredVMFromVirtualMachine(t *testing.T) {
	const (
		validResourceID = "/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/vm-name"
		subscriptionID  = "11111111-1111-1111-1111-111111111111"
		resourceGroup   = "rg"
	)

	for _, tc := range []struct {
		desc        string
		vm          *armcompute.VirtualMachine
		assertError require.ErrorAssertionFunc
		assertVM    require.ValueAssertionFunc
	}{
		{
			desc: "vm with all fields populated",
			vm: &armcompute.VirtualMachine{
				ID:       to.Ptr(validResourceID),
				Name:     to.Ptr("vm-name"),
				Location: to.Ptr("eastus"),
				Properties: &armcompute.VirtualMachineProperties{
					VMID: to.Ptr("22222222-2222-2222-2222-222222222222"),
				},
				Tags: map[string]*string{
					"env":  to.Ptr("prod"),
					"team": to.Ptr("infra"),
				},
			},
			assertError: require.NoError,
			assertVM: func(t require.TestingT, val any, _ ...any) {
				require.NotNil(t, val)
				vm, ok := val.(*DiscoveredVM)
				require.Truef(t, ok, "expected *DiscoveredVM, got %T", val)
				require.Equal(t, &DiscoveredVM{
					ID:             validResourceID,
					VMID:           "22222222-2222-2222-2222-222222222222",
					Name:           "vm-name",
					Location:       "eastus",
					SubscriptionID: subscriptionID,
					ResourceGroup:  resourceGroup,
					Tags: map[string]string{
						"env":  "prod",
						"team": "infra",
					},
				}, vm)
			},
		},
		{
			desc: "vm with nil Properties does not panic and has empty VMID",
			vm: &armcompute.VirtualMachine{
				ID:       to.Ptr(validResourceID),
				Name:     to.Ptr("vm-name"),
				Location: to.Ptr("eastus"),
			},
			assertError: require.NoError,
			assertVM: func(t require.TestingT, val any, _ ...any) {
				require.NotNil(t, val)
				vm, ok := val.(*DiscoveredVM)
				require.Truef(t, ok, "expected *DiscoveredVM, got %T", val)
				require.Empty(t, vm.VMID)
				require.Equal(t, validResourceID, vm.ID)
				require.Equal(t, subscriptionID, vm.SubscriptionID)
				require.Equal(t, resourceGroup, vm.ResourceGroup)
			},
		},
		{
			desc: "vm without tags returns empty (not nil) map",
			vm: &armcompute.VirtualMachine{
				ID:   to.Ptr(validResourceID),
				Name: to.Ptr("vm-name"),
			},
			assertError: require.NoError,
			assertVM: func(t require.TestingT, val any, _ ...any) {
				require.NotNil(t, val)
				vm, ok := val.(*DiscoveredVM)
				require.Truef(t, ok, "expected *DiscoveredVM, got %T", val)
				require.NotNil(t, vm.Tags)
				require.Empty(t, vm.Tags)
			},
		},
		{
			desc: "vm not part of a Scale Set has empty Scale Set fields",
			vm: &armcompute.VirtualMachine{
				ID:   to.Ptr(validResourceID),
				Name: to.Ptr("vm-name"),
			},
			assertError: require.NoError,
			assertVM: func(t require.TestingT, val any, _ ...any) {
				require.NotNil(t, val)
				vm, ok := val.(*DiscoveredVM)
				require.Truef(t, ok, "expected *DiscoveredVM, got %T", val)
				require.Empty(t, vm.UniformScaleSetName)
				require.Empty(t, vm.ScaleSetVMInstanceID)
			},
		},
		{
			desc:        "nil vm returns error",
			vm:          nil,
			assertError: require.Error,
			assertVM:    require.Nil,
		},
		{
			desc: "invalid resource ID returns error",
			vm: &armcompute.VirtualMachine{
				ID:   to.Ptr("not-a-valid-arm-id"),
				Name: to.Ptr("vm-name"),
			},
			assertError: require.Error,
			assertVM:    require.Nil,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			vm, err := discoveredVMFromVirtualMachine(tc.vm)
			tc.assertError(t, err)
			tc.assertVM(t, vm)
		})
	}
}

func TestDiscoveredVMFromVirtualMachineScaleSetVM(t *testing.T) {
	const (
		validResourceID = "/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg/providers/Microsoft.Compute/virtualMachineScaleSets/vmss/virtualMachines/0"
		subscriptionID  = "11111111-1111-1111-1111-111111111111"
		resourceGroup   = "rg"
		scaleSetName    = "vmss"
	)

	for _, tc := range []struct {
		desc           string
		vm             *armcompute.VirtualMachineScaleSetVM
		scaleSetName   string
		resourceGroup  string
		subscriptionID string
		assertError    require.ErrorAssertionFunc
		assertVM       require.ValueAssertionFunc
	}{
		{
			desc: "scale set vm with all fields populated",
			vm: &armcompute.VirtualMachineScaleSetVM{
				ID:         to.Ptr(validResourceID),
				Name:       to.Ptr("vmss_0"),
				Location:   to.Ptr("eastus"),
				InstanceID: to.Ptr("0"),
				Properties: &armcompute.VirtualMachineScaleSetVMProperties{
					VMID: to.Ptr("22222222-2222-2222-2222-222222222222"),
				},
				Tags: map[string]*string{
					"env":  to.Ptr("prod"),
					"team": to.Ptr("infra"),
				},
			},
			scaleSetName:   scaleSetName,
			resourceGroup:  resourceGroup,
			subscriptionID: subscriptionID,
			assertError:    require.NoError,
			assertVM: func(t require.TestingT, val any, _ ...any) {
				require.NotNil(t, val)
				vm, ok := val.(*DiscoveredVM)
				require.Truef(t, ok, "expected *DiscoveredVM, got %T", val)
				require.Equal(t, &DiscoveredVM{
					ID:                   validResourceID,
					VMID:                 "22222222-2222-2222-2222-222222222222",
					Name:                 "vmss_0",
					Location:             "eastus",
					SubscriptionID:       subscriptionID,
					ResourceGroup:        resourceGroup,
					UniformScaleSetName:  scaleSetName,
					ScaleSetVMInstanceID: "0",
					Tags: map[string]string{
						"env":  "prod",
						"team": "infra",
					},
				}, vm)
			},
		},
		{
			desc: "scale set vm with nil Properties does not panic and has empty VMID",
			vm: &armcompute.VirtualMachineScaleSetVM{
				ID:         to.Ptr(validResourceID),
				Name:       to.Ptr("vmss_0"),
				Location:   to.Ptr("eastus"),
				InstanceID: to.Ptr("0"),
			},
			scaleSetName:   scaleSetName,
			resourceGroup:  resourceGroup,
			subscriptionID: subscriptionID,
			assertError:    require.NoError,
			assertVM: func(t require.TestingT, val any, _ ...any) {
				require.NotNil(t, val)
				vm, ok := val.(*DiscoveredVM)
				require.Truef(t, ok, "expected *DiscoveredVM, got %T", val)
				require.Empty(t, vm.VMID)
				require.Equal(t, validResourceID, vm.ID)
				require.Equal(t, scaleSetName, vm.UniformScaleSetName)
				require.Equal(t, "0", vm.ScaleSetVMInstanceID)
			},
		},
		{
			desc: "scale set vm without tags returns empty (not nil) map",
			vm: &armcompute.VirtualMachineScaleSetVM{
				ID:         to.Ptr(validResourceID),
				Name:       to.Ptr("vmss_0"),
				InstanceID: to.Ptr("0"),
			},
			scaleSetName:   scaleSetName,
			resourceGroup:  resourceGroup,
			subscriptionID: subscriptionID,
			assertError:    require.NoError,
			assertVM: func(t require.TestingT, val any, _ ...any) {
				require.NotNil(t, val)
				vm, ok := val.(*DiscoveredVM)
				require.Truef(t, ok, "expected *DiscoveredVM, got %T", val)
				require.NotNil(t, vm.Tags)
				require.Empty(t, vm.Tags)
			},
		},
		{
			desc: "uses caller-provided subscription, resource group, and scale set name without parsing ID",
			vm: &armcompute.VirtualMachineScaleSetVM{
				ID:         to.Ptr("not-a-valid-arm-id"),
				Name:       to.Ptr("vmss_0"),
				InstanceID: to.Ptr("0"),
			},
			scaleSetName:   scaleSetName,
			resourceGroup:  resourceGroup,
			subscriptionID: subscriptionID,
			assertError:    require.NoError,
			assertVM: func(t require.TestingT, val any, _ ...any) {
				require.NotNil(t, val)
				vm, ok := val.(*DiscoveredVM)
				require.Truef(t, ok, "expected *DiscoveredVM, got %T", val)
				require.Equal(t, subscriptionID, vm.SubscriptionID)
				require.Equal(t, resourceGroup, vm.ResourceGroup)
				require.Equal(t, scaleSetName, vm.UniformScaleSetName)
			},
		},
		{
			desc:           "nil vm returns error",
			vm:             nil,
			scaleSetName:   scaleSetName,
			resourceGroup:  resourceGroup,
			subscriptionID: subscriptionID,
			assertError:    require.Error,
			assertVM:       require.Nil,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			vm, err := discoveredVMFromVirtualMachineScaleSetVM(tc.vm, tc.scaleSetName, tc.resourceGroup, tc.subscriptionID)
			tc.assertError(t, err)
			tc.assertVM(t, vm)
		})
	}
}
