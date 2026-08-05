// Package communication provides executable migration boundaries for the
// legacy communication Kafka/websocket JSON envelopes. Typed gRPC or other
// protobuf-native transports use protojson separately; legacy envelopes must use
// these encoding/json-compatible functions so int64 identifiers remain JSON
// numbers instead of protojson strings.
package communication

import (
	"encoding/json"
	"errors"
	"time"

	chatv1 "github.com/4ubak/cg-proto/gen/go/communication/chat"
	notificationv1 "github.com/4ubak/cg-proto/gen/go/communication/notification"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var errNilLegacyEnvelope = errors.New("legacy communication envelope is nil")

type legacyChatEvent struct {
	MessageID               string                 `json:"message_id"`
	ChatID                  string                 `json:"chat_id"`
	SenderID                int64                  `json:"sender_id"`
	MessageType             string                 `json:"message_type"`
	RecipientID             int64                  `json:"recipient_id"`
	RecipientUserIDs        []int64                `json:"recipient_user_ids,omitempty"`
	TargetApps              []string               `json:"target_apps"`
	RecipientOrgID          string                 `json:"recipient_org_id"`
	RecipientApp            chatv1.ChatApp         `json:"recipient_app,omitempty"`
	RecipientPerspective    chatv1.ChatPerspective `json:"recipient_perspective,omitempty"`
	RecipientOrganizationID string                 `json:"recipient_organization_id,omitempty"`
	ContextType             string                 `json:"context_type"`
	ContextID               string                 `json:"context_id"`
	RecipientScope          *chatv1.ChatScope      `json:"recipient_scope,omitempty"`
}

// MarshalLegacyChatEvent preserves the encoding/json wire used by chat.events.
func MarshalLegacyChatEvent(event *chatv1.ChatRealtimeEventPayload) ([]byte, error) {
	if event == nil {
		return nil, errNilLegacyEnvelope
	}
	return json.Marshal(legacyChatEvent{
		MessageID:               event.GetMessageId(),
		ChatID:                  event.GetChatId(),
		SenderID:                event.GetSenderId(),
		MessageType:             event.GetMessageType(),
		RecipientID:             event.GetRecipientId(),
		RecipientUserIDs:        event.GetRecipientUserIds(),
		TargetApps:              event.GetTargetApps(),
		RecipientOrgID:          event.GetRecipientOrgId(),
		RecipientApp:            event.GetRecipientApp(),
		RecipientPerspective:    event.GetRecipientPerspective(),
		RecipientOrganizationID: event.GetRecipientOrganizationId(),
		ContextType:             event.GetContextType(),
		ContextID:               event.GetContextId(),
		RecipientScope:          event.GetRecipientScope(),
	})
}

// UnmarshalLegacyChatEvent accepts the numeric encoding/json envelope consumed
// by websocket-service and both mobile/partner BFF event consumers.
func UnmarshalLegacyChatEvent(data []byte, event *chatv1.ChatRealtimeEventPayload) error {
	if event == nil {
		return errNilLegacyEnvelope
	}
	var wire legacyChatEvent
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	*event = chatv1.ChatRealtimeEventPayload{
		MessageId:               wire.MessageID,
		ChatId:                  wire.ChatID,
		SenderId:                wire.SenderID,
		MessageType:             wire.MessageType,
		RecipientId:             wire.RecipientID,
		RecipientUserIds:        wire.RecipientUserIDs,
		TargetApps:              wire.TargetApps,
		RecipientOrgId:          wire.RecipientOrgID,
		RecipientApp:            wire.RecipientApp,
		RecipientPerspective:    wire.RecipientPerspective,
		RecipientOrganizationId: wire.RecipientOrganizationID,
		ContextType:             wire.ContextType,
		ContextId:               wire.ContextID,
		RecipientScope:          wire.RecipientScope,
	}
	return nil
}

type legacyPushEvent struct {
	UserID                  int64                                  `json:"user_id"`
	EventType               string                                 `json:"event_type"`
	Title                   string                                 `json:"title"`
	Body                    string                                 `json:"body"`
	Data                    map[string]string                      `json:"data,omitempty"`
	Priority                string                                 `json:"priority"`
	DedupKey                string                                 `json:"dedup_key"`
	TargetApps              []string                               `json:"target_apps,omitempty"`
	Category                string                                 `json:"category,omitempty"`
	TypedTargetApps         []notificationv1.NotificationApp       `json:"typed_target_apps,omitempty"`
	RecipientPerspective    notificationv1.NotificationPerspective `json:"recipient_perspective,omitempty"`
	RecipientOrganizationID string                                 `json:"recipient_organization_id,omitempty"`
	ContextType             string                                 `json:"context_type,omitempty"`
	ContextID               string                                 `json:"context_id,omitempty"`
	TypedCategory           notificationv1.NotificationCategory    `json:"typed_category,omitempty"`
	RecipientScopes         []*notificationv1.NotificationScope    `json:"recipient_scopes,omitempty"`
}

// MarshalLegacyPushEvent preserves notification.push numeric user_id and exact
// snake_case encoding/json keys.
func MarshalLegacyPushEvent(event *notificationv1.PushEventPayload) ([]byte, error) {
	if event == nil {
		return nil, errNilLegacyEnvelope
	}
	return json.Marshal(legacyPushEvent{
		UserID:                  event.GetUserId(),
		EventType:               event.GetEventType(),
		Title:                   event.GetTitle(),
		Body:                    event.GetBody(),
		Data:                    event.GetData(),
		Priority:                event.GetPriority(),
		DedupKey:                event.GetDedupKey(),
		TargetApps:              event.GetTargetApps(),
		Category:                event.GetCategory(),
		TypedTargetApps:         event.GetTypedTargetApps(),
		RecipientPerspective:    event.GetRecipientPerspective(),
		RecipientOrganizationID: event.GetRecipientOrganizationId(),
		ContextType:             event.GetContextType(),
		ContextID:               event.GetContextId(),
		TypedCategory:           event.GetTypedCategory(),
		RecipientScopes:         event.GetRecipientScopes(),
	})
}

// UnmarshalLegacyPushEvent accepts the current encoding/json producer and
// consumer envelope without accepting protojson's quoted int64 representation.
func UnmarshalLegacyPushEvent(data []byte, event *notificationv1.PushEventPayload) error {
	if event == nil {
		return errNilLegacyEnvelope
	}
	var wire legacyPushEvent
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	*event = notificationv1.PushEventPayload{
		UserId:                  wire.UserID,
		EventType:               wire.EventType,
		Title:                   wire.Title,
		Body:                    wire.Body,
		Data:                    wire.Data,
		Priority:                wire.Priority,
		DedupKey:                wire.DedupKey,
		TargetApps:              wire.TargetApps,
		Category:                wire.Category,
		TypedTargetApps:         wire.TypedTargetApps,
		RecipientPerspective:    wire.RecipientPerspective,
		RecipientOrganizationId: wire.RecipientOrganizationID,
		ContextType:             wire.ContextType,
		ContextId:               wire.ContextID,
		TypedCategory:           wire.TypedCategory,
		RecipientScopes:         wire.RecipientScopes,
	}
	return nil
}

type legacyRealtimeNotification struct {
	UserID                  int64                                  `json:"user_id"`
	ID                      string                                 `json:"id"`
	Type                    string                                 `json:"type"`
	Category                string                                 `json:"category"`
	Title                   string                                 `json:"title"`
	Body                    string                                 `json:"body"`
	Data                    map[string]string                      `json:"data"`
	IsRead                  bool                                   `json:"is_read"`
	CreatedAt               string                                 `json:"created_at,omitempty"`
	RecipientApp            notificationv1.NotificationApp         `json:"recipient_app,omitempty"`
	RecipientPerspective    notificationv1.NotificationPerspective `json:"recipient_perspective,omitempty"`
	RecipientOrganizationID string                                 `json:"recipient_organization_id,omitempty"`
	ContextType             string                                 `json:"context_type,omitempty"`
	ContextID               string                                 `json:"context_id,omitempty"`
	TypedType               notificationv1.NotificationType        `json:"typed_type,omitempty"`
	TypedCategory           notificationv1.NotificationCategory    `json:"typed_category,omitempty"`
	RecipientScope          *notificationv1.NotificationScope      `json:"recipient_scope,omitempty"`
}

// MarshalLegacyRealtimeNotification preserves notification.new's numeric
// user_id and RFC3339 created_at representation.
func MarshalLegacyRealtimeNotification(event *notificationv1.RealtimeNotificationEventPayload) ([]byte, error) {
	if event == nil {
		return nil, errNilLegacyEnvelope
	}
	createdAt := ""
	if event.GetCreatedAt() != nil {
		createdAt = event.GetCreatedAt().AsTime().UTC().Format(time.RFC3339Nano)
	}
	return json.Marshal(legacyRealtimeNotification{
		UserID:                  event.GetUserId(),
		ID:                      event.GetId(),
		Type:                    event.GetType(),
		Category:                event.GetCategory(),
		Title:                   event.GetTitle(),
		Body:                    event.GetBody(),
		Data:                    event.GetData(),
		IsRead:                  event.GetIsRead(),
		CreatedAt:               createdAt,
		RecipientApp:            event.GetRecipientApp(),
		RecipientPerspective:    event.GetRecipientPerspective(),
		RecipientOrganizationID: event.GetRecipientOrganizationId(),
		ContextType:             event.GetContextType(),
		ContextID:               event.GetContextId(),
		TypedType:               event.GetTypedType(),
		TypedCategory:           event.GetTypedCategory(),
		RecipientScope:          event.GetRecipientScope(),
	})
}

// UnmarshalLegacyRealtimeNotification decodes the current notification.new
// encoding/json envelope.
func UnmarshalLegacyRealtimeNotification(data []byte, event *notificationv1.RealtimeNotificationEventPayload) error {
	if event == nil {
		return errNilLegacyEnvelope
	}
	var wire legacyRealtimeNotification
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	var createdAt *timestamppb.Timestamp
	if wire.CreatedAt != "" {
		parsed, err := time.Parse(time.RFC3339Nano, wire.CreatedAt)
		if err != nil {
			return err
		}
		createdAt = timestamppb.New(parsed)
	}
	*event = notificationv1.RealtimeNotificationEventPayload{
		UserId:                  wire.UserID,
		Id:                      wire.ID,
		Type:                    wire.Type,
		Category:                wire.Category,
		Title:                   wire.Title,
		Body:                    wire.Body,
		Data:                    wire.Data,
		IsRead:                  wire.IsRead,
		CreatedAt:               createdAt,
		RecipientApp:            wire.RecipientApp,
		RecipientPerspective:    wire.RecipientPerspective,
		RecipientOrganizationId: wire.RecipientOrganizationID,
		ContextType:             wire.ContextType,
		ContextId:               wire.ContextID,
		TypedType:               wire.TypedType,
		TypedCategory:           wire.TypedCategory,
		RecipientScope:          wire.RecipientScope,
	}
	return nil
}
