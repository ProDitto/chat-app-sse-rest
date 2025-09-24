package domain

import "time"

type EventType string

const (
	UserJoined      EventType = "user_joined"
	UserLeft        EventType = "user_left"
	UsernameChanged EventType = "username_changed"
	Message         EventType = "message"
	TypingStart     EventType = "typing_start"
	TypingStop      EventType = "typing_stop"
	Heartbeat       EventType = "heartbeat"
)

type Event struct {
	EventID    string                 `json:"eventId"`
	Timestamp  time.Time              `json:"timestamp"`
	Type       EventType              `json:"type"`
	FromUserID string                 `json:"fromUserId,omitempty"`
	ToUserID   string                 `json:"toUserId,omitempty"`
	Payload    map[string]interface{} `json:"payload"`
}

