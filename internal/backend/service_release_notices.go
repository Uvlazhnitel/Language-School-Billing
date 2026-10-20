package backend

import (
	"context"
	"langschool/internal/app/releasenotice"
)

func (s *Service) ReleaseNoticeVisit(ctx context.Context, userID int) (*releasenotice.Result, error) {
	return releasenotice.New(s.rt.DB.Ent).Visit(ctx, userID)
}
func (s *Service) ReleaseNoticeDismiss(ctx context.Context, userID int, releaseID string) error {
	return releasenotice.New(s.rt.DB.Ent).Dismiss(ctx, userID, releaseID)
}
