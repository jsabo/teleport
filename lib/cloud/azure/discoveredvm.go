/*
 * Teleport
 * Copyright (C) 2026 Gravitational, Inc.
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
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/gravitational/trace"
)

// DiscoveredVM describes an Azure virtual machine returned by discovery.
type DiscoveredVM struct {
	// ID is the ARM resource ID, e.g. "/subscriptions/.../virtualMachines/foo".
	ID string
	// SubscriptionID is the Azure subscription containing the VM, e.g. "11111111-1111-1111-1111-111111111111".
	SubscriptionID string
	// UniformScaleSetName is the name of the Virtual Machine Scale Set. Empty string if the VM is not part of a uniform Scale Set.
	UniformScaleSetName string
	// ScaleSetVMInstanceID is the instance ID of the Virtual Machine Scale Set VM. Empty string if the VM is not part of a uniform Scale Set.
	// This is a unique identifier for the VM within its Scale Set, e.g. "0", "1".
	ScaleSetVMInstanceID string
	// Name is the VM's display name, e.g. "teleport-agent-01".
	Name string
	// VMID is Azure's unique identifier for the VM, e.g. "22222222-2222-2222-2222-222222222222".
	VMID string
	// Location is the Azure region containing the VM, e.g. "eastus".
	Location string
	// ResourceGroup is the Azure resource group containing the VM, e.g. "teleport-rg".
	ResourceGroup string
	// Tags are the VM tags, e.g. {"env": "prod"}. Empty map (not nil) when the VM has no tags.
	Tags map[string]string
}

func discoveredVMFromVirtualMachine(vm *armcompute.VirtualMachine) (*DiscoveredVM, error) {
	if vm == nil {
		return nil, trace.BadParameter("vm cannot be nil")
	}

	var vmid string
	if vm.Properties != nil {
		vmid = StringVal(vm.Properties.VMID)
	}

	// The ARM resource ID should never fail to parse, unless Azure changes the contract.
	resourceMetadata, err := arm.ParseResourceID(StringVal(vm.ID))
	if err != nil {
		return nil, trace.BadParameter("failed to parse Virtual Machine resource ID %q: %v", StringVal(vm.ID), err)
	}

	return &DiscoveredVM{
		ID:             StringVal(vm.ID),
		VMID:           vmid,
		Name:           StringVal(vm.Name),
		Location:       StringVal(vm.Location),
		SubscriptionID: resourceMetadata.SubscriptionID,
		ResourceGroup:  resourceMetadata.ResourceGroupName,
		Tags:           ConvertTags(vm.Tags),
	}, nil
}

func discoveredVMFromVirtualMachineScaleSetVM(vm *armcompute.VirtualMachineScaleSetVM, scaleSetName, resourceGroup, subscriptionID string) (*DiscoveredVM, error) {
	if vm == nil {
		return nil, trace.BadParameter("vm cannot be nil")
	}

	var vmid string
	if vm.Properties != nil {
		vmid = StringVal(vm.Properties.VMID)
	}

	return &DiscoveredVM{
		ID:                   StringVal(vm.ID),
		VMID:                 vmid,
		Name:                 StringVal(vm.Name),
		Location:             StringVal(vm.Location),
		SubscriptionID:       subscriptionID,
		ResourceGroup:        resourceGroup,
		Tags:                 ConvertTags(vm.Tags),
		UniformScaleSetName:  scaleSetName,
		ScaleSetVMInstanceID: StringVal(vm.InstanceID),
	}, nil
}
