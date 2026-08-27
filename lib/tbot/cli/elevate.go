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
	"log/slog"
	"time"

	"github.com/gravitational/trace"

	"github.com/gravitational/teleport/lib/tbot/bot"
	"github.com/gravitational/teleport/lib/tbot/config"
	"github.com/gravitational/teleport/lib/tbot/internal"
	"github.com/gravitational/teleport/lib/tbot/services/identity"
	"github.com/gravitational/teleport/lib/tbot/services/k8s"
)

// ElevateCommand implements `tbot elevate`: obtain credentials for roles the
// bot does not hold, through a just-in-time access request.
//
// Elevation is event-driven — something needs fixing now — rather than
// timer-driven, so this runs once and exits instead of joining the renewal
// loop. A long-running bot configured this way would file a new request, and
// block on a human, every renewal interval.
type ElevateCommand struct {
	*sharedStartArgs
	*sharedDestinationArgs
	*genericMutatorHandler

	Roles       []string
	Reason      string
	Reviewers   []string
	MaxDuration time.Duration
	Timeout     time.Duration
	TTL         time.Duration
	Cluster     string
	KubeCluster string
}

// NewElevateCommand initializes the command and flags for `tbot elevate`.
func NewElevateCommand(app KingpinClause, action MutatorAction) *ElevateCommand {
	cmd := app.Command("elevate",
		"Request elevated roles via an access request and write the resulting credentials.")

	c := &ElevateCommand{}
	c.sharedDestinationArgs = newSharedDestinationArgs(cmd)
	c.sharedStartArgs = newSharedStartArgs(cmd)
	c.genericMutatorHandler = newGenericMutatorHandler(cmd, c, action)

	cmd.Flag("roles", "Roles to request. The bot's role must allow requesting them.").
		Required().StringsVar(&c.Roles)
	cmd.Flag("reason", "Reason shown to reviewers. Some clusters require one.").
		StringVar(&c.Reason)
	cmd.Flag("reviewers", "Reviewers to suggest for the request.").
		StringsVar(&c.Reviewers)
	cmd.Flag("max-duration", "How long the granted access may last.").
		DurationVar(&c.MaxDuration)
	cmd.Flag("ttl", "Lifetime of the issued credentials.").
		DurationVar(&c.TTL)
	cmd.Flag("timeout", "How long to wait for a reviewer before giving up.").
		DurationVar(&c.Timeout)
	cmd.Flag("cluster", "Issue the identity for this leaf cluster.").
		StringVar(&c.Cluster)
	cmd.Flag("kube-cluster", "Write a kubeconfig for this Kubernetes cluster instead of an identity file.").
		StringVar(&c.KubeCluster)

	return c
}

func (c *ElevateCommand) ApplyConfig(cfg *config.BotConfig, l *slog.Logger) error {
	if err := c.sharedStartArgs.ApplyConfig(cfg, l); err != nil {
		return trace.Wrap(err)
	}

	dest, err := c.BuildDestination()
	if err != nil {
		return trace.Wrap(err)
	}

	// Always one-shot: see the type comment. Set here rather than asking the
	// caller for --oneshot, so the renewal trap cannot be configured at all.
	cfg.Oneshot = true

	request := &internal.AccessRequestConfig{
		Roles:       c.Roles,
		Reason:      c.Reason,
		Reviewers:   c.Reviewers,
		MaxDuration: c.MaxDuration,
		Timeout:     c.Timeout,
	}

	// A kubeconfig is what most remediation tooling consumes, so producing one
	// directly saves the caller assembling it from an identity file.
	var lifetime bot.CredentialLifetime
	if c.TTL > 0 {
		// Both fields must be set together, but nothing is ever renewed here:
		// the command exits once the credentials are written.
		lifetime.TTL = c.TTL
		lifetime.RenewalInterval = c.TTL
	}

	if c.KubeCluster != "" {
		cfg.Services = append(cfg.Services, &k8s.OutputV2Config{
			Destination:        dest,
			Selectors:          []*k8s.KubernetesSelector{{Name: c.KubeCluster}},
			AccessRequest:      request,
			CredentialLifetime: lifetime,
		})
		return nil
	}

	cfg.Services = append(cfg.Services, &identity.OutputConfig{
		Destination:        dest,
		Cluster:            c.Cluster,
		AccessRequest:      request,
		CredentialLifetime: lifetime,
	})

	return nil
}
