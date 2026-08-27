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

package cli

import (
	"errors"

	"github.com/gravitational/teleport/lib/tbot/internal"
)

// ExitCodeAccessRequestDenied is returned when a reviewer denied the access
// request made by `tbot elevate`.
//
// A denial is a decision rather than a malfunction, so it is reported
// separately from the general failure code. Callers that gate a privileged
// action on the command — where the exit status *is* the decision — need to
// tell "the human said no" apart from "this did not work".
const ExitCodeAccessRequestDenied = 2

// IsAccessRequestDenied reports whether the error was caused by a reviewer
// denying an access request.
func IsAccessRequestDenied(err error) bool {
	return errors.Is(err, internal.ErrAccessRequestDenied)
}
