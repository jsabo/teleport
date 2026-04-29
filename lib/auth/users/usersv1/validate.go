package usersv1

import (
	"github.com/gravitational/trace"

	userspb "github.com/gravitational/teleport/api/gen/proto/go/teleport/users/v1"
	"github.com/gravitational/teleport/lib/auth/authclient"
	"github.com/gravitational/teleport/lib/defaults"
)

func validateResetUserRequest(r *userspb.ResetUserRequest) error {
	if r.Name == "" {
		return trace.BadParameter("user name can't be empty")
	}

	if r.Ttl == nil {
		return trace.BadParameter("TTL can't be nil")
	}
	if r.Ttl.AsDuration() < 0 {
		return trace.BadParameter("TTL can't be negative")
	}

	switch r.Type {
	case authclient.UserTokenTypeResetPasswordInvite:
		if r.Ttl.AsDuration() > defaults.MaxSignupTokenTTL {
			return trace.BadParameter(
				"maximum token TTL for the user invitation flow is %v hours",
				defaults.MaxSignupTokenTTL)
		}

	case authclient.UserTokenTypeResetPassword:
		if r.Ttl.AsDuration() > defaults.MaxChangePasswordTokenTTL {
			return trace.BadParameter(
				"maximum token TTL for the password reset flow is %v hours",
				defaults.MaxChangePasswordTokenTTL)
		}

	default:
		return trace.BadParameter("unknown user token request type(%v)", r.Type)
	}

	return nil
}
