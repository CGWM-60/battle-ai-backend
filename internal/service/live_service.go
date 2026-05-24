package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"
	"cgwm/battle/internal/repository"

	"gorm.io/datatypes"
)

type LiveService struct {
	live *repository.LiveRepository
}

func NewLiveService(live *repository.LiveRepository) *LiveService {
	return &LiveService{live: live}
}

func (s *LiveService) AppendBattleEvent(ctx context.Context, battleID uint, eventType, authorType, authorName string, payload any) {
	sessions, err := s.live.ListSessionsByBattle(ctx, battleID)
	if err != nil {
		return
	}
	for _, session := range sessions {
		_ = s.AppendEvent(ctx, session.Id, eventType, authorType, authorName, payload)
	}
}

func (s *LiveService) AppendEvent(ctx context.Context, sessionID uint, eventType, authorType, authorName string, payload any) error {
	sequence, err := s.live.NextEventSequence(ctx, sessionID)
	if err != nil {
		return err
	}
	data, _ := json.Marshal(payload)
	event := &models.LiveEvent{
		LiveSessionID: sessionID,
		Sequence:      sequence,
		EventType:     eventType,
		AuthorType:    authorType,
		AuthorName:    authorName,
		Payload:       datatypes.JSON(data),
	}
	if err := s.live.AppendEvent(ctx, event); err != nil {
		return err
	}
	now := time.Now()
	return s.live.UpdateSessionFields(ctx, sessionID, map[string]any{"last_event_at": &now})
}

func (s *LiveService) EndSessionsByBattle(ctx context.Context, battleID uint) error {
	sessions, err := s.live.ListSessionsByBattle(ctx, battleID)
	if err != nil {
		return err
	}

	var lastErr error
	for _, session := range sessions {
		if err := s.EndSession(ctx, session.Id, session.ChannelKey); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

func (s *LiveService) EndSession(ctx context.Context, sessionID uint, channelKey string) error {
	now := time.Now()
	if err := s.AppendEvent(ctx, sessionID, constants.LiveEventTypeStatus, constants.AuthorTypeSystem, constants.AuthorTypeSystem, map[string]any{
		"status":  constants.LiveStatusEnded,
		"channel": channelKey,
		"message": "live session ended",
	}); err != nil {
		return err
	}

	return s.live.UpdateSessionFields(ctx, sessionID, map[string]any{
		"status":        constants.LiveStatusEnded,
		"ended_at":      &now,
		"last_event_at": &now,
	})
}

func (s *LiveService) HistoryByChannel(ctx context.Context, channel string, ownerID uint, after int, limit int) (*models.LiveSession, []models.LiveEvent, error) {
	session, err := s.live.GetSessionOwnedByChannel(ctx, channel, ownerID)
	if err != nil {
		return nil, nil, err
	}
	if !session.AllowReplay && session.Status == constants.LiveStatusEnded {
		return nil, nil, fmt.Errorf("replay disabled for this channel")
	}
	events, err := s.live.ListEventsAfter(ctx, session.Id, after, limit)
	return session, events, err
}
