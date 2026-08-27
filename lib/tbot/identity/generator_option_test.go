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

package identity

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gravitational/teleport/api/client/proto"
)

// applyOpts runs the given options and returns the resulting certificate
// request, mirroring what Generate does before sending it.
func applyOpts(t *testing.T, req *proto.UserCertsRequest, options ...GenerateOption) *proto.UserCertsRequest {
	t.Helper()
	o := &generateOpts{}
	for _, fn := range options {
		fn(o)
	}
	for _, fn := range o.requestModifiers {
		fn(req)
	}
	return req
}

func TestWithAccessRequests(t *testing.T) {
	t.Run("sets access requests", func(t *testing.T) {
		req := applyOpts(t, &proto.UserCertsRequest{}, WithAccessRequests([]string{"req-1", "req-2"}))
		require.Equal(t, []string{"req-1", "req-2"}, req.AccessRequests)
	})

	t.Run("clears role impersonation", func(t *testing.T) {
		// The auth server rejects a request carrying both role requests and
		// access requests, and the bot's default roles are applied before
		// options run, so the option has to clear them rather than assume they
		// are unset.
		req := &proto.UserCertsRequest{
			RoleRequests:    []string{"some-default-role"},
			UseRoleRequests: true,
		}
		applyOpts(t, req, WithAccessRequests([]string{"req-1"}))

		require.Equal(t, []string{"req-1"}, req.AccessRequests)
		require.Empty(t, req.RoleRequests, "role requests must not accompany access requests")
		require.False(t, req.UseRoleRequests, "UseRoleRequests must not accompany access requests")
	})

	t.Run("leaves unrelated fields alone", func(t *testing.T) {
		req := &proto.UserCertsRequest{
			Username:          "bot-sre-remediator",
			KubernetesCluster: "k8s-eks",
		}
		applyOpts(t, req, WithAccessRequests([]string{"req-1"}))

		require.Equal(t, "bot-sre-remediator", req.Username)
		require.Equal(t, "k8s-eks", req.KubernetesCluster)
	})
}
