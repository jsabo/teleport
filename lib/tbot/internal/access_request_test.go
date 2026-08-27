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

package internal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccessRequestConfig_CheckAndSetDefaults(t *testing.T) {
	tests := []struct {
		name    string
		in      *AccessRequestConfig
		want    *AccessRequestConfig
		wantErr string
	}{
		{
			name: "valid",
			in:   &AccessRequestConfig{Roles: []string{"editor"}, Timeout: time.Minute},
			want: &AccessRequestConfig{Roles: []string{"editor"}, Timeout: time.Minute},
		},
		{
			// Without a bound wait, an unattended request hangs the bot
			// indefinitely.
			name: "timeout defaults",
			in:   &AccessRequestConfig{Roles: []string{"editor"}},
			want: &AccessRequestConfig{Roles: []string{"editor"}, Timeout: defaultAccessRequestTimeout},
		},
		{
			name:    "roles required",
			in:      &AccessRequestConfig{},
			wantErr: "access_request: roles is required",
		},
		{
			name:    "negative timeout",
			in:      &AccessRequestConfig{Roles: []string{"editor"}, Timeout: -time.Second},
			wantErr: "access_request: timeout must not be negative",
		},
		{
			name:    "negative max duration",
			in:      &AccessRequestConfig{Roles: []string{"editor"}, MaxDuration: -time.Second},
			wantErr: "access_request: max_duration must not be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.in.CheckAndSetDefaults()
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, tt.in)
		})
	}
}

// A denial is a decision, not a failure to retry. Callers rely on being able to
// tell the two apart, so the sentinel must survive being wrapped.
func TestErrAccessRequestDenied_IsDistinguishable(t *testing.T) {
	require.Error(t, ErrAccessRequestDenied)
	assert.Contains(t, ErrAccessRequestDenied.Error(), "denied")
}
