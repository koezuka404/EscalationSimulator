package websocket

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"escalator/entity"
	"escalator/redis"
	"escalator/usecase"
)

type envelope struct {
	Origin      string          `json:"origin"`
	Staff       bool            `json:"staff"`
	Admins      bool            `json:"admins"`
	ApplicantID string          `json:"applicant_id"`
	UserID      string          `json:"user_id"`
	Body        json.RawMessage `json:"body"`
}

type RemoteNotices struct {
	hub    *Hub
	bus    *redis.Bus
	prefix string
}

func NewRemoteNotices(hub *Hub, bus *redis.Bus, prefix string) *RemoteNotices {
	prefix = strings.TrimSpace(prefix)
	if prefix != "" && !strings.HasSuffix(prefix, ":") {
		prefix += ":"
	}
	return &RemoteNotices{hub: hub, bus: bus, prefix: prefix}
}

//このサーバーの画面へ届け、他のサーバーへも渡す
func (r *RemoteNotices) Send(message usecase.Message) {
	r.hub.Send(message)
	if r.bus == nil || r.prefix == "" {
		return
	}
	payload, err := json.Marshal(envelope{
		Origin:      r.bus.Origin(),
		Staff:       message.Staff,
		Admins:      message.Admins,
		ApplicantID: message.ApplicantID,
		UserID:      message.UserID,
		Body:        message.Body,
	})
	if err != nil {
		log.Printf("他のサーバーへ渡す知らせを作れませんでした: %v", err)
		return
	}
	for _, channel := range noticeChannels(r.prefix, message) {
		if err := r.bus.Publish(context.Background(), channel, payload); err != nil {
			log.Printf("他のサーバーへ知らせを渡せませんでした: %v", err)
		}
	}
}

//他のサーバーの知らせを、このサーバーの画面へ届ける
func (r *RemoteNotices) Run(ctx context.Context) {
	if r.bus == nil || r.prefix == "" {
		return
	}
	channels := []string{r.prefix + "applicant", r.prefix + "agent", r.prefix + "admin"}
	r.bus.Listen(ctx, channels, func(channel string, payload []byte) {
		var received envelope
		if err := json.Unmarshal(payload, &received); err != nil {
			log.Printf("他のサーバーからの知らせを読めませんでした: %v", err)
			return
		}
		if received.Origin == r.bus.Origin() {
			return
		}
		role, ok := roleFromChannel(r.prefix, channel)
		if !ok {
			return
		}
		r.hub.deliver(role, usecase.Message{
			Staff:       received.Staff,
			Admins:      received.Admins,
			ApplicantID: received.ApplicantID,
			UserID:      received.UserID,
			Body:        received.Body,
		})
	})
}

func noticeChannels(prefix string, message usecase.Message) []string {
	seen := map[string]struct{}{}
	add := func(role string) {
		if role == "" {
			return
		}
		seen[prefix+role] = struct{}{}
	}
	if message.UserID != "" {
		add("applicant")
		add("agent")
		add("admin")
	}
	if message.Staff {
		add("agent")
		add("admin")
	}
	if message.Admins {
		add("admin")
	}
	if message.ApplicantID != "" {
		add("applicant")
	}
	channels := make([]string, 0, len(seen))
	for channel := range seen {
		channels = append(channels, channel)
	}
	return channels
}

func roleFromChannel(prefix, channel string) (entity.Role, bool) {
	role := strings.TrimPrefix(channel, prefix)
	switch entity.Role(role) {
	case entity.RoleApplicant, entity.RoleAgent, entity.RoleAdmin:
		return entity.Role(role), true
	default:
		return "", false
	}
}
