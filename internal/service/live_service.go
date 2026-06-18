package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"cgwm/battle/internal/app/constants"
	"cgwm/battle/internal/models"
	nexuscache "cgwm/battle/internal/nexus_game/cache"
	"cgwm/battle/internal/repository"

	"gorm.io/datatypes"
)

type LiveService struct {
	live  *repository.LiveRepository
	cache *nexuscache.ResponseCache
}

func NewLiveService(live *repository.LiveRepository) *LiveService {
	return &LiveService{live: live}
}

func NewLiveServiceWithCache(live *repository.LiveRepository, cache *nexuscache.ResponseCache) *LiveService {
	return &LiveService{live: live, cache: cache}
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
	if err := s.live.UpdateSessionFields(ctx, sessionID, map[string]any{"last_event_at": &now}); err != nil {
		return err
	}
	s.cache.InvalidateNamespace(ctx, "live")
	return nil
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
	if err := s.live.UpdateSessionFields(ctx, sessionID, map[string]any{
		"status":        constants.LiveStatusEnded,
		"ended_at":      &now,
		"last_event_at": &now,
	}); err != nil {
		return err
	}

	_ = s.AppendEvent(ctx, sessionID, constants.LiveEventTypeStatus, constants.AuthorTypeSystem, constants.AuthorTypeSystem, map[string]any{
		"status":  constants.LiveStatusEnded,
		"channel": channelKey,
		"message": "live session ended",
	})
	s.cache.InvalidateNamespace(ctx, "live")
	return nil
}

func (s *LiveService) HistoryByChannel(ctx context.Context, channel string, ownerID uint, after int, limit int) (*models.LiveSession, []models.LiveEvent, error) {
	key := fmt.Sprintf("history:owner:%d:channel:%s:after:%d:limit:%d", ownerID, channel, after, limit)
	var cached struct {
		Session models.LiveSession `json:"session"`
		Events  []models.LiveEvent `json:"events"`
	}
	if s.cache.GetJSON(ctx, "live", key, &cached) {
		return &cached.Session, cached.Events, nil
	}
	session, err := s.live.GetSessionOwnedByChannel(ctx, channel, ownerID)
	if err != nil {
		return nil, nil, err
	}
	if !session.AllowReplay && session.Status == constants.LiveStatusEnded {
		return nil, nil, fmt.Errorf("replay disabled for this channel")
	}
	events, err := s.live.ListEventsAfter(ctx, session.Id, after, limit)
	if err != nil {
		return nil, nil, err
	}
	s.cache.SetJSON(ctx, "live", key, struct {
		Session models.LiveSession `json:"session"`
		Events  []models.LiveEvent `json:"events"`
	}{Session: *session, Events: events}, time.Second)
	return session, events, nil
}
