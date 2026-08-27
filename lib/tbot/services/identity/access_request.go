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
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/gravitational/trace"

	apiclient "github.com/gravitational/teleport/api/client"
	"github.com/gravitational/teleport/api/types"
)

// ErrAccessRequestDenied is returned when a reviewer denies the bot's access
// request. Callers distinguish this from an operational failure: a denial is a
// decision, not an error to retry.
var ErrAccessRequestDenied = &trace.AccessDeniedError{Message: "access request was denied"}

// requestAccess creates an access request for the configured roles and blocks
// until a reviewer resolves it, returning the approved request's ID.
//
// The bot's own identity is used, which is what makes this possible: the auth
// server refuses AccessRequests from role-impersonated identities
// ("impersonated user can not request new roles"), and every credential tbot
// hands to a workload is impersonated. Only the bot itself can make this call.
func requestAccess(
	ctx context.Context,
	clt *apiclient.Client,
	botUser string,
	cfg *AccessRequestConfig,
	log *slog.Logger,
) (string, error) {
	req, err := types.NewAccessRequest(uuid.NewString(), botUser, cfg.Roles...)
	if err != nil {
		return "", trace.Wrap(err, "building access request")
	}
	if cfg.Reason != "" {
		req.SetRequestReason(cfg.Reason)
	}
	if len(cfg.Reviewers) > 0 {
		req.SetSuggestedReviewers(cfg.Reviewers)
	}
	if cfg.MaxDuration > 0 {
		req.SetMaxDuration(time.Now().UTC().Add(cfg.MaxDuration))
	}

	created, err := clt.CreateAccessRequestV2(ctx, req)
	if err != nil {
		return "", trace.Wrap(err, "creating access request")
	}

	// Emitted before blocking so an operator can find the request to approve,
	// and so automation can correlate it with the audit log.
	log.InfoContext(ctx,
		"Created access request, waiting for a reviewer",
		"request_id", created.GetName(),
		"roles", cfg.Roles,
		"timeout", cfg.Timeout,
	)

	resolved, err := awaitResolution(ctx, clt, created, cfg.Timeout)
	if err != nil {
		return "", trace.Wrap(err)
	}

	switch resolved.GetState() {
	case types.RequestState_APPROVED:
		log.InfoContext(ctx,
			"Access request approved",
			"request_id", resolved.GetName(),
			"reason", resolved.GetResolveReason(),
		)
		return resolved.GetName(), nil
	case types.RequestState_DENIED:
		// Distinct from an error: a human said no.
		return "", trace.Wrap(ErrAccessRequestDenied, "request %s denied: %s",
			resolved.GetName(), resolved.GetResolveReason())
	default:
		return "", trace.BadParameter("access request %s in unexpected state %s",
			resolved.GetName(), resolved.GetState())
	}
}

// awaitResolution blocks until the request leaves the pending state, the
// timeout elapses, or the context is cancelled.
//
// A watcher is used rather than polling so approval is acted on immediately;
// the request's current state is read once after the watcher is established to
// close the race where it is resolved before the watch begins.
func awaitResolution(
	ctx context.Context,
	clt *apiclient.Client,
	req types.AccessRequest,
	timeout time.Duration,
) (types.AccessRequest, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	filter := types.AccessRequestFilter{
		User: req.GetUser(),
		ID:   req.GetName(),
	}
	watcher, err := clt.NewWatcher(ctx, types.Watch{
		Name: "tbot-await-access-request",
		Kinds: []types.WatchKind{{
			Kind:   types.KindAccessRequest,
			Filter: filter.IntoMap(),
		}},
	})
	if err != nil {
		return nil, trace.Wrap(err, "watching access request")
	}
	defer watcher.Close()

	select {
	case event := <-watcher.Events():
		if event.Type != types.OpInit {
			return nil, trace.BadParameter("expected OpInit while establishing watch, got %v", event.Type)
		}
	case <-watcher.Done():
		return nil, trace.Wrap(watcher.Error())
	case <-ctx.Done():
		return nil, trace.Wrap(timeoutErr(ctx, timeout))
	}

	// The request may already have been resolved between creation and the
	// watch being ready.
	current, err := getRequest(ctx, clt, req.GetName())
	if err != nil {
		return nil, trace.Wrap(err)
	}
	if !current.GetState().IsPending() {
		return current, nil
	}

	for {
		select {
		case event := <-watcher.Events():
			switch event.Type {
			case types.OpPut:
				updated, ok := event.Resource.(types.AccessRequest)
				if !ok {
					return nil, trace.BadParameter("unexpected resource type %T", event.Resource)
				}
				if !updated.GetState().IsPending() {
					return updated, nil
				}
			case types.OpDelete:
				return nil, trace.NotFound("access request %s expired or was deleted before review",
					req.GetName())
			}
		case <-watcher.Done():
			return nil, trace.Wrap(watcher.Error())
		case <-ctx.Done():
			return nil, trace.Wrap(timeoutErr(ctx, timeout))
		}
	}
}

func getRequest(ctx context.Context, clt *apiclient.Client, id string) (types.AccessRequest, error) {
	reqs, err := clt.GetAccessRequests(ctx, types.AccessRequestFilter{ID: id})
	if err != nil {
		return nil, trace.Wrap(err)
	}
	if len(reqs) == 0 {
		return nil, trace.NotFound("access request %s not found", id)
	}
	return reqs[0], nil
}

func timeoutErr(ctx context.Context, timeout time.Duration) error {
	if ctx.Err() == context.DeadlineExceeded {
		return trace.LimitExceeded("no reviewer resolved the access request within %s", timeout)
	}
	return ctx.Err()
}
