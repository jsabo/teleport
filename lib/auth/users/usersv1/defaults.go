package usersv1

import (
	"google.golang.org/protobuf/types/known/durationpb"

	userspb "github.com/gravitational/teleport/api/gen/proto/go/teleport/users/v1"
	"github.com/gravitational/teleport/lib/auth/authclient"
	"github.com/gravitational/teleport/lib/defaults"
)

func setResetUserRequestDefaults(r *userspb.ResetUserRequest) {
	if r.Type == "" {
		r.Type = authclient.UserTokenTypeResetPassword
	}

	if r.GetTtl().AsDuration() == 0 {
		switch r.Type {
		case authclient.UserTokenTypeResetPasswordInvite:
			r.Ttl = durationpb.New(defaults.SignupTokenTTL)

		case authclient.UserTokenTypeResetPassword:
			r.Ttl = durationpb.New(defaults.ChangePasswordTokenTTL)

		default:
			// This is invalid, but we are not validating here, so set up any non-nil
			// value just to reduce a risk of panic.
			r.Ttl = durationpb.New(0)
		}
	}
}
