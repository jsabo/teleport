/*
 * Teleport
 * Copyright (C) 2023  Gravitational, Inc.
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
	"context"
	"fmt"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/gravitational/trace"
	"github.com/stretchr/testify/require"

	"github.com/gravitational/teleport/api/types"
)

const (
	testSubID = "11111111-1111-1111-1111-111111111111"
	rgName    = "rg1"

	vmResourceID = "/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg1/providers/Microsoft.Compute/virtualMachines/vm1"

	vmssResourceID   = "/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg1/providers/Microsoft.Compute/virtualMachineScaleSets/vmss1"
	vmssVMResourceID = "/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg1/providers/Microsoft.Compute/virtualMachineScaleSets/vmss1/virtualMachines/0"
)

func TestGetVirtualMachine(t *testing.T) {
	ctx := context.Background()
	validResourceID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/vm"

	for _, tc := range []struct {
		desc        string
		resourceID  string
		client      *ARMComputeMock
		assertError require.ErrorAssertionFunc
		assertVM    require.ValueAssertionFunc
	}{
		{
			desc:       "vm with valid user identities",
			resourceID: validResourceID,
			client: &ARMComputeMock{
				GetResult: armcompute.VirtualMachine{
					ID:   to.Ptr(validResourceID),
					Name: to.Ptr("name"),
					Identity: &armcompute.VirtualMachineIdentity{
						PrincipalID: to.Ptr("system assigned"),
						Type:        to.Ptr(armcompute.ResourceIdentityTypeSystemAssigned),
						UserAssignedIdentities: map[string]*armcompute.UserAssignedIdentitiesValue{
							"identity1": {},
							"identity2": {},
						},
					},
				},
			},
			assertError: require.NoError,
			assertVM: func(t require.TestingT, val any, _ ...any) {
				require.NotNil(t, val)
				vm, ok := val.(*VirtualMachine)
				require.Truef(t, ok, "expected *VirtualMachine, got %T", val)
				require.Equal(t, vm.ID, validResourceID)
				require.Equal(t, "name", vm.Name)
				require.ElementsMatch(t, []Identity{
					{ResourceID: "system assigned"},
					{ResourceID: "identity1"},
					{ResourceID: "identity2"},
				}, vm.Identities)
			},
		},
		{
			desc:       "vm without identity",
			resourceID: validResourceID,
			client: &ARMComputeMock{
				GetResult: armcompute.VirtualMachine{
					ID:   to.Ptr(validResourceID),
					Name: to.Ptr("name"),
				},
			},
			assertError: require.NoError,
			assertVM: func(t require.TestingT, val any, _ ...any) {
				require.NotNil(t, val)
				vm, ok := val.(*VirtualMachine)
				require.Truef(t, ok, "expected *VirtualMachine, got %T", val)
				require.Equal(t, vm.ID, validResourceID)
				require.Equal(t, "name", vm.Name)
				require.Empty(t, vm.Identities)
			},
		},
		{
			desc:       "vm with only user managed identities",
			resourceID: validResourceID,
			client: &ARMComputeMock{
				GetResult: armcompute.VirtualMachine{
					ID:   to.Ptr(validResourceID),
					Name: to.Ptr("name"),
					Identity: &armcompute.VirtualMachineIdentity{
						UserAssignedIdentities: map[string]*armcompute.UserAssignedIdentitiesValue{
							"identity1": {},
							"identity2": {},
						},
					},
				},
			},
			assertError: require.NoError,
			assertVM: func(t require.TestingT, val any, _ ...any) {
				require.NotNil(t, val)
				vm, ok := val.(*VirtualMachine)
				require.Truef(t, ok, "expected *VirtualMachine, got %T", val)
				require.Equal(t, vm.ID, validResourceID)
				require.Equal(t, "name", vm.Name)
				require.ElementsMatch(t, []Identity{
					{ResourceID: "identity1"},
					{ResourceID: "identity2"},
				}, vm.Identities)
			},
		},
		{
			desc:        "invalid resource ID",
			resourceID:  "random-id",
			client:      &ARMComputeMock{},
			assertError: require.Error,
			assertVM:    require.Nil,
		},
		{
			desc:       "client error",
			resourceID: validResourceID,
			client: &ARMComputeMock{
				GetErr: fmt.Errorf("client error"),
			},
			assertError: require.Error,
			assertVM:    require.Nil,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			vmClient := NewVirtualMachinesClientByAPI(VirtualMachinesClientConfig{
				VirtualMachineAPI: tc.client,
			})

			vm, err := vmClient.Get(ctx, tc.resourceID)
			tc.assertError(t, err)
			tc.assertVM(t, vm)
		})
	}
}

func TestGetScaleSetVirtualMachine(t *testing.T) {
	ctx := context.Background()
	validResourceID := "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Compute/virtualMachineScaleSets/vmss/virtualMachines/0"

	for _, tc := range []struct {
		desc        string
		resourceID  string
		client      *ARMScaleSetVMsMock
		assertError require.ErrorAssertionFunc
		assertVM    require.ValueAssertionFunc
	}{
		{
			desc:       "vm with valid user identities",
			resourceID: validResourceID,
			client: &ARMScaleSetVMsMock{
				GetResult: armcompute.VirtualMachineScaleSetVM{
					ID:   to.Ptr(validResourceID),
					Name: to.Ptr("name"),
					Identity: &armcompute.VirtualMachineIdentity{
						PrincipalID: to.Ptr("system assigned"),
						Type:        to.Ptr(armcompute.ResourceIdentityTypeSystemAssigned),
						UserAssignedIdentities: map[string]*armcompute.UserAssignedIdentitiesValue{
							"identity1": {},
							"identity2": {},
						},
					},
				},
			},
			assertError: require.NoError,
			assertVM: func(t require.TestingT, val any, _ ...any) {
				require.NotNil(t, val)
				vm, ok := val.(*VirtualMachine)
				require.Truef(t, ok, "expected *VirtualMachine, got %T", val)
				require.Equal(t, vm.ID, validResourceID)
				require.Equal(t, "name", vm.Name)
				require.ElementsMatch(t, []Identity{
					{ResourceID: "system assigned"},
					{ResourceID: "identity1"},
					{ResourceID: "identity2"},
				}, vm.Identities)
			},
		},
		{
			desc:       "vm without identity",
			resourceID: validResourceID,
			client: &ARMScaleSetVMsMock{
				GetResult: armcompute.VirtualMachineScaleSetVM{
					ID:   to.Ptr(validResourceID),
					Name: to.Ptr("name"),
				},
			},
			assertError: require.NoError,
			assertVM: func(t require.TestingT, val any, _ ...any) {
				require.NotNil(t, val)
				vm, ok := val.(*VirtualMachine)
				require.Truef(t, ok, "expected *VirtualMachine, got %T", val)
				require.Equal(t, vm.ID, validResourceID)
				require.Equal(t, "name", vm.Name)
				require.Empty(t, vm.Identities)
			},
		},
		{
			desc:       "vm with only user managed identities",
			resourceID: validResourceID,
			client: &ARMScaleSetVMsMock{
				GetResult: armcompute.VirtualMachineScaleSetVM{
					ID:   to.Ptr(validResourceID),
					Name: to.Ptr("name"),
					Identity: &armcompute.VirtualMachineIdentity{
						UserAssignedIdentities: map[string]*armcompute.UserAssignedIdentitiesValue{
							"identity1": {},
							"identity2": {},
						},
					},
				},
			},
			assertError: require.NoError,
			assertVM: func(t require.TestingT, val any, _ ...any) {
				require.NotNil(t, val)
				vm, ok := val.(*VirtualMachine)
				require.Truef(t, ok, "expected *VirtualMachine, got %T", val)
				require.Equal(t, vm.ID, validResourceID)
				require.Equal(t, "name", vm.Name)
				require.ElementsMatch(t, []Identity{
					{ResourceID: "identity1"},
					{ResourceID: "identity2"},
				}, vm.Identities)
			},
		},
		{
			desc:       "client error",
			resourceID: validResourceID,
			client: &ARMScaleSetVMsMock{
				GetErr: fmt.Errorf("client error"),
			},
			assertError: require.Error,
			assertVM:    require.Nil,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			vmClient := NewVirtualMachinesClientByAPI(VirtualMachinesClientConfig{
				ScaleSetVMsAPI: tc.client,
			})

			vm, err := vmClient.Get(ctx, tc.resourceID)
			tc.assertError(t, err)
			tc.assertVM(t, vm)
		})
	}
}

func TestListVirtualMachines(t *testing.T) {
	t.Parallel()
	mockAPI := &ARMComputeMock{
		VirtualMachines: map[string][]*armcompute.VirtualMachine{
			"rg1": {
				{ID: to.Ptr("vm1")},
				{ID: to.Ptr("vm2")},
			},
			"rg2": {
				{ID: to.Ptr("vm3")},
				{ID: to.Ptr("vm4")},
			},
		},
	}
	tests := []struct {
		name          string
		resourceGroup string
		wantIDs       []string
	}{
		{
			name:          "existing resource group",
			resourceGroup: "rg1",
			wantIDs:       []string{"vm1", "vm2"},
		},
		{
			name:          "nonexistant resource group",
			resourceGroup: "rgfake",
			wantIDs:       []string{},
		},
		{
			name:          "all resource groups",
			resourceGroup: types.Wildcard,
			wantIDs:       []string{"vm1", "vm2", "vm3", "vm4"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := NewVirtualMachinesClientByAPI(VirtualMachinesClientConfig{
				VirtualMachineAPI: mockAPI,
			})

			vms, err := client.ListVirtualMachines(context.Background(), tc.resourceGroup)
			require.NoError(t, err)
			var vmIDs []string
			for _, vm := range vms {
				vmIDs = append(vmIDs, *vm.ID)
			}
			require.ElementsMatch(t, tc.wantIDs, vmIDs)
		})
	}
}

// discoveredVMIDs returns the VMID field from each DiscoveredVM.
// VMID is the Azure-assigned UUID for the VM (e.g. "22222222-..."), distinct from
// the ARM resource ID. It is used throughout the tests as a stable identifier
// to verify which VMs were returned.
func discoveredVMIDs(vms []*DiscoveredVM) []string {
	ids := make([]string, 0, len(vms))
	for _, vm := range vms {
		ids = append(ids, vm.VMID)
	}
	return ids
}

func TestListAllVirtualMachines(t *testing.T) {
	t.Parallel()

	// A well-formed regular VM whose ARM ID is parseable.
	regularVM := &armcompute.VirtualMachine{
		ID:       to.Ptr(vmResourceID),
		Name:     to.Ptr("vm1"),
		Location: to.Ptr("eastus"),
		Properties: &armcompute.VirtualMachineProperties{
			VMID: to.Ptr("vm-uuid-1"),
		},
	}

	// A well-formed uniform scale set with a parseable ARM ID.
	uniformScaleSet := &armcompute.VirtualMachineScaleSet{
		ID:   to.Ptr(vmssResourceID),
		Name: to.Ptr("vmss1"),
	}

	// A VM belonging to the scale set above.
	scaleSetVM := armcompute.VirtualMachineScaleSetVM{
		ID:         to.Ptr(vmssVMResourceID),
		Name:       to.Ptr("vmss1_0"),
		Location:   to.Ptr("eastus"),
		InstanceID: to.Ptr("0"),
		Properties: &armcompute.VirtualMachineScaleSetVMProperties{
			VMID: to.Ptr("vmss-vm-uuid-1"),
		},
	}

	for _, tc := range []struct {
		desc           string
		vmAPI          *ARMComputeMock
		scaleSetsAPI   *ARMScaleSetsMock
		scaleSetVMsAPI *ARMScaleSetVMsMock
		resourceGroup  string
		wantVMIDs      []string
	}{
		{
			desc: "regular VMs only, no scale sets",
			vmAPI: &ARMComputeMock{
				VirtualMachines: map[string][]*armcompute.VirtualMachine{
					rgName: {regularVM},
				},
			},
			scaleSetsAPI:   &ARMScaleSetsMock{},
			scaleSetVMsAPI: &ARMScaleSetVMsMock{},
			resourceGroup:  rgName,
			wantVMIDs:      []string{"vm-uuid-1"},
		},
		{
			desc:  "uniform VMSS VMs only, no regular VMs",
			vmAPI: &ARMComputeMock{VirtualMachines: map[string][]*armcompute.VirtualMachine{}},
			scaleSetsAPI: &ARMScaleSetsMock{
				ScaleSetRecords: []*armcompute.VirtualMachineScaleSet{uniformScaleSet},
			},
			scaleSetVMsAPI: &ARMScaleSetVMsMock{GetResult: scaleSetVM},
			resourceGroup:  rgName,
			wantVMIDs:      []string{"vmss-vm-uuid-1"},
		},
		{
			desc: "combines regular VMs and uniform VMSS VMs",
			vmAPI: &ARMComputeMock{
				VirtualMachines: map[string][]*armcompute.VirtualMachine{
					rgName: {regularVM},
				},
			},
			scaleSetsAPI: &ARMScaleSetsMock{
				ScaleSetRecords: []*armcompute.VirtualMachineScaleSet{uniformScaleSet},
			},
			scaleSetVMsAPI: &ARMScaleSetVMsMock{GetResult: scaleSetVM},
			resourceGroup:  rgName,
			wantVMIDs:      []string{"vm-uuid-1", "vmss-vm-uuid-1"},
		},
		{
			desc: "flexible scale set is skipped by the VMSS lister",
			vmAPI: &ARMComputeMock{
				VirtualMachines: map[string][]*armcompute.VirtualMachine{
					rgName: {regularVM},
				},
			},
			scaleSetsAPI: &ARMScaleSetsMock{
				ScaleSetRecords: []*armcompute.VirtualMachineScaleSet{
					{
						ID:   to.Ptr(vmssResourceID),
						Name: to.Ptr("vmss1"),
						Properties: &armcompute.VirtualMachineScaleSetProperties{
							OrchestrationMode: to.Ptr(armcompute.OrchestrationModeFlexible),
						},
					},
				},
			},
			// scaleSetVMsAPI would return a VM, but it must never be called for
			// a flexible scale set.
			scaleSetVMsAPI: &ARMScaleSetVMsMock{GetResult: scaleSetVM},
			resourceGroup:  rgName,
			wantVMIDs:      []string{"vm-uuid-1"},
		},
		{
			// When listing VMs within a scale set fails, the scale set is skipped
			// (inner loop breaks) but processing continues. Regular VMs are unaffected.
			desc:  "scale set VM listing error skips that scale set gracefully",
			vmAPI: &ARMComputeMock{VirtualMachines: map[string][]*armcompute.VirtualMachine{}},
			scaleSetsAPI: &ARMScaleSetsMock{
				ScaleSetRecords: []*armcompute.VirtualMachineScaleSet{uniformScaleSet},
			},
			scaleSetVMsAPI: &ARMScaleSetVMsMock{ListErr: trace.NotFound("vms not found")},
			resourceGroup:  rgName,
			wantVMIDs:      []string{},
		},
		{
			desc:  "scale set with invalid ARM ID is skipped, valid scale sets still processed",
			vmAPI: &ARMComputeMock{VirtualMachines: map[string][]*armcompute.VirtualMachine{}},
			scaleSetsAPI: &ARMScaleSetsMock{
				ScaleSetRecords: []*armcompute.VirtualMachineScaleSet{
					{ID: to.Ptr("not-a-valid-arm-id"), Name: to.Ptr("bad")},
					uniformScaleSet,
				},
			},
			scaleSetVMsAPI: &ARMScaleSetVMsMock{GetResult: scaleSetVM},
			resourceGroup:  rgName,
			wantVMIDs:      []string{"vmss-vm-uuid-1"},
		},
		{
			desc: "wildcard resource group returns VMs across all resource groups",
			vmAPI: &ARMComputeMock{
				VirtualMachines: map[string][]*armcompute.VirtualMachine{
					"rg1": {
						{
							ID:   to.Ptr("/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg1/providers/Microsoft.Compute/virtualMachines/vm1"),
							Name: to.Ptr("vm1"),
							Properties: &armcompute.VirtualMachineProperties{
								VMID: to.Ptr("uuid-1"),
							},
						},
					},
					"rg2": {
						{
							ID:   to.Ptr("/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg2/providers/Microsoft.Compute/virtualMachines/vm2"),
							Name: to.Ptr("vm2"),
							Properties: &armcompute.VirtualMachineProperties{
								VMID: to.Ptr("uuid-2"),
							},
						},
					},
				},
			},
			scaleSetsAPI:   &ARMScaleSetsMock{},
			scaleSetVMsAPI: &ARMScaleSetVMsMock{},
			resourceGroup:  types.Wildcard,
			wantVMIDs:      []string{"uuid-1", "uuid-2"},
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			client := NewVirtualMachinesClientByAPI(VirtualMachinesClientConfig{
				VirtualMachineAPI: tc.vmAPI,
				ScaleSetsAPI:      tc.scaleSetsAPI,
				ScaleSetVMsAPI:    tc.scaleSetVMsAPI,
			})

			vms, err := client.ListAllVirtualMachines(t.Context(), tc.resourceGroup)
			require.NoError(t, err)
			require.ElementsMatch(t, tc.wantVMIDs, discoveredVMIDs(vms))
		})
	}
}

// TestListAllVirtualMachines_VMSSFields verifies that DiscoveredVMs returned for
// uniform scale set VMs are populated with the correct scale set metadata.
func TestListAllVirtualMachines_VMSSFields(t *testing.T) {
	t.Parallel()

	scaleSetVM := armcompute.VirtualMachineScaleSetVM{
		ID:         to.Ptr(vmssVMResourceID),
		Name:       to.Ptr("vmss1_0"),
		Location:   to.Ptr("eastus"),
		InstanceID: to.Ptr("0"),
		Properties: &armcompute.VirtualMachineScaleSetVMProperties{
			VMID: to.Ptr("vmss-vm-uuid-1"),
		},
	}

	client := NewVirtualMachinesClientByAPI(VirtualMachinesClientConfig{
		VirtualMachineAPI: &ARMComputeMock{VirtualMachines: map[string][]*armcompute.VirtualMachine{}},
		ScaleSetsAPI: &ARMScaleSetsMock{
			ScaleSetRecords: []*armcompute.VirtualMachineScaleSet{
				{ID: to.Ptr(vmssResourceID), Name: to.Ptr("vmss1")},
			},
		},
		ScaleSetVMsAPI: &ARMScaleSetVMsMock{GetResult: scaleSetVM},
	})

	vms, err := client.ListAllVirtualMachines(t.Context(), rgName)
	require.NoError(t, err)
	require.Len(t, vms, 1)

	vm := vms[0]
	require.Equal(t, "vmss1", vm.UniformScaleSetName)
	require.Equal(t, "0", vm.ScaleSetVMInstanceID)
	require.Equal(t, testSubID, vm.SubscriptionID)
	require.Equal(t, rgName, vm.ResourceGroup)
	require.Equal(t, "vmss-vm-uuid-1", vm.VMID)
	require.Equal(t, "eastus", vm.Location)
}

func TestVmScaleSetIsFlexible(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		desc       string
		properties *armcompute.VirtualMachineScaleSetProperties
		want       bool
	}{
		{
			desc:       "nil properties is not flexible (treated as Uniform)",
			properties: nil,
			want:       false,
		},
		{
			desc: "flexible orchestration mode",
			properties: &armcompute.VirtualMachineScaleSetProperties{
				OrchestrationMode: to.Ptr(armcompute.OrchestrationModeFlexible),
			},
			want: true,
		},
		{
			desc: "uniform orchestration mode",
			properties: &armcompute.VirtualMachineScaleSetProperties{
				OrchestrationMode: to.Ptr(armcompute.OrchestrationModeUniform),
			},
			want: false,
		},
		{
			desc:       "properties without orchestration mode is not flexible",
			properties: &armcompute.VirtualMachineScaleSetProperties{},
			want:       false,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			require.Equal(t, tc.want, vmScaleSetIsFlexible(tc.properties))
		})
	}
}

func TestRunCommandRequestIsUniformVMSS(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		desc string
		req  RunCommandRequest
		want bool
	}{
		{
			desc: "regular VM, no scale set fields",
			req:  RunCommandRequest{VMName: "vm1", ResourceGroup: "rg1"},
			want: false,
		},
		{
			desc: "both ScaleSetName and InstanceID set",
			req:  RunCommandRequest{VMName: "vmss1_0", UniformScaleSetName: "vmss1", ScaleSetVMInstanceID: "0"},
			want: true,
		},
		{
			desc: "only ScaleSetName, no InstanceID",
			req:  RunCommandRequest{VMName: "vmss1_0", UniformScaleSetName: "vmss1"},
			want: false,
		},
		{
			desc: "only InstanceID, no ScaleSetName",
			req:  RunCommandRequest{VMName: "vmss1_0", ScaleSetVMInstanceID: "0"},
			want: false,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			require.Equal(t, tc.want, tc.req.isUniformVMSS())
		})
	}
}

func TestCommandResultFromInstanceView(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		desc         string
		properties   *armcompute.VirtualMachineRunCommandProperties
		wantErr      bool
		wantState    string
		wantExitCode int32
		wantStdOut   string
		wantStdErr   string
		wantFailure  bool
	}{
		{
			desc:    "nil properties returns error",
			wantErr: true,
		},
		{
			desc:       "nil InstanceView returns error",
			properties: &armcompute.VirtualMachineRunCommandProperties{},
			wantErr:    true,
		},
		{
			desc: "succeeded command with zero exit code",
			properties: &armcompute.VirtualMachineRunCommandProperties{
				InstanceView: &armcompute.VirtualMachineRunCommandInstanceView{
					ExecutionState: to.Ptr(armcompute.ExecutionStateSucceeded),
					ExitCode:       to.Ptr(int32(0)),
					Output:         to.Ptr("install ok"),
					Error:          to.Ptr(""),
				},
			},
			wantState:    string(armcompute.ExecutionStateSucceeded),
			wantExitCode: 0,
			wantStdOut:   "install ok",
			wantStdErr:   "",
			wantFailure:  false,
		},
		{
			desc: "failed state with non-zero exit code",
			properties: &armcompute.VirtualMachineRunCommandProperties{
				InstanceView: &armcompute.VirtualMachineRunCommandInstanceView{
					ExecutionState: to.Ptr(armcompute.ExecutionStateFailed),
					ExitCode:       to.Ptr(int32(1)),
					Output:         to.Ptr(""),
					Error:          to.Ptr("command not found"),
				},
			},
			wantState:    string(armcompute.ExecutionStateFailed),
			wantExitCode: 1,
			wantStdErr:   "command not found",
			wantFailure:  true,
		},
		{
			desc: "succeeded state with non-zero exit code is still a failure",
			properties: &armcompute.VirtualMachineRunCommandProperties{
				InstanceView: &armcompute.VirtualMachineRunCommandInstanceView{
					ExecutionState: to.Ptr(armcompute.ExecutionStateSucceeded),
					ExitCode:       to.Ptr(int32(127)),
					Output:         to.Ptr(""),
					Error:          to.Ptr("script exited with code 127"),
				},
			},
			wantState:    string(armcompute.ExecutionStateSucceeded),
			wantExitCode: 127,
			wantStdErr:   "script exited with code 127",
			wantFailure:  true,
		},
		{
			desc: "nil pointer fields in InstanceView default to zero values",
			properties: &armcompute.VirtualMachineRunCommandProperties{
				InstanceView: &armcompute.VirtualMachineRunCommandInstanceView{},
			},
			wantState:    "",
			wantExitCode: 0,
			wantFailure:  true,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			result, err := commandResultFromInstanceView(tc.properties)
			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, result)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			require.Equal(t, tc.wantState, result.ExecutionState)
			require.Equal(t, tc.wantExitCode, result.ExitCode)
			require.Equal(t, tc.wantStdOut, result.StdOut)
			require.Equal(t, tc.wantStdErr, result.StdErr)
			require.Equal(t, tc.wantFailure, result.Failure())
		})
	}
}
