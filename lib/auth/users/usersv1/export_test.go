package usersv1

import "context"

func (s *Service) ResetPassword(ctx context.Context, username string) error {
	return s.resetPassword(ctx, username)
}
