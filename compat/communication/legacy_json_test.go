package communication

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	chatv1 "github.com/4ubak/cg-proto/gen/go/communication/chat"
	notificationv1 "github.com/4ubak/cg-proto/gen/go/communication/notification"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const exactLargeID int64 = 9007199254740993

func TestLegacyChatJSONKeepsAllConsumerIDsNumeric(t *testing.T) {
	event := &chatv1.ChatRealtimeEventPayload{
		MessageId:        "message-1",
		ChatId:           "chat-1",
		SenderId:         exactLargeID,
		MessageType:      "text",
		RecipientId:      exactLargeID - 1,
		RecipientUserIds: []int64{exactLargeID - 1, exactLargeID - 2},
		TargetApps:       []string{"client"},
		ContextType:      "request",
		ContextId:        "request-1",
		RecipientScope: &chatv1.ChatScope{
			App:               chatv1.ChatApp_CHAT_APP_PRO,
			Perspective:       chatv1.ChatPerspective_CHAT_PERSPECTIVE_SELLER_ORG,
			OrganizationId:    "org-7",
			MembershipVersion: 17,
		},
		Recipients: []*chatv1.ChatRecipient{
			{UserId: exactLargeID - 1, Scope: &chatv1.ChatScope{App: chatv1.ChatApp_CHAT_APP_PRO, Perspective: chatv1.ChatPerspective_CHAT_PERSPECTIVE_SELLER_ORG, OrganizationId: "org-7", MembershipVersion: 17}},
			{UserId: exactLargeID - 2, Scope: &chatv1.ChatScope{App: chatv1.ChatApp_CHAT_APP_PRO, Perspective: chatv1.ChatPerspective_CHAT_PERSPECTIVE_SELLER_ORG, OrganizationId: "org-7", MembershipVersion: 18}},
		},
	}

	wire, err := MarshalLegacyChatEvent(event)
	if err != nil {
		t.Fatalf("marshal legacy chat event: %v", err)
	}
	assertJSONKeys(t, wire, "message_id", "chat_id", "sender_id", "message_type", "recipient_id", "recipient_user_ids", "target_apps", "recipient_org_id", "context_type", "context_id", "recipients")
	assertNumericJSONField(t, wire, "sender_id", exactLargeID)
	assertNumericJSONField(t, wire, "recipient_id", exactLargeID-1)
	assertNumericJSONArray(t, wire, "recipient_user_ids", []int64{exactLargeID - 1, exactLargeID - 2})
	assertNestedMembershipVersion(t, wire, "recipient_scope", 17)
	assertBoundChatRecipientsNumeric(t, wire, []int64{exactLargeID - 1, exactLargeID - 2}, []int64{17, 18})

	var mobileConsumer struct {
		MessageID        string  `json:"message_id"`
		ChatID           string  `json:"chat_id"`
		SenderID         int64   `json:"sender_id"`
		RecipientID      int64   `json:"recipient_id"`
		RecipientUserIDs []int64 `json:"recipient_user_ids"`
	}
	if err := json.Unmarshal(wire, &mobileConsumer); err != nil {
		t.Fatalf("current mobile/websocket consumer decode: %v", err)
	}
	if mobileConsumer.SenderID != exactLargeID || mobileConsumer.RecipientID != exactLargeID-1 ||
		!reflect.DeepEqual(mobileConsumer.RecipientUserIDs, []int64{exactLargeID - 1, exactLargeID - 2}) {
		t.Fatalf("consumer IDs changed: %+v", mobileConsumer)
	}

	var roundTrip chatv1.ChatRealtimeEventPayload
	if err := UnmarshalLegacyChatEvent(wire, &roundTrip); err != nil {
		t.Fatalf("unmarshal legacy chat event: %v", err)
	}
	if roundTrip.GetSenderId() != event.GetSenderId() || roundTrip.GetRecipientId() != event.GetRecipientId() ||
		!reflect.DeepEqual(roundTrip.GetRecipientUserIds(), event.GetRecipientUserIds()) || len(roundTrip.GetRecipients()) != 2 ||
		roundTrip.GetRecipients()[1].GetScope().GetMembershipVersion() != 18 {
		t.Fatalf("chat round trip changed integer IDs: %+v", &roundTrip)
	}
}

func assertBoundChatRecipientsNumeric(t *testing.T, data []byte, wantUsers, wantVersions []int64) {
	t.Helper()
	values := decodeNumbers(t, data)
	items, ok := values["recipients"].([]any)
	if !ok || len(items) != len(wantUsers) || len(wantUsers) != len(wantVersions) {
		t.Fatalf("recipients = %#v, want %d entries", values["recipients"], len(wantUsers))
	}
	for i, item := range items {
		recipient, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("recipients[%d] type = %T", i, item)
		}
		userID, ok := recipient["user_id"].(json.Number)
		if !ok {
			t.Fatalf("recipients[%d].user_id type = %T, want number", i, recipient["user_id"])
		}
		gotUser, err := userID.Int64()
		if err != nil || gotUser != wantUsers[i] {
			t.Fatalf("recipients[%d].user_id = %q (%v), want %d", i, userID, err, wantUsers[i])
		}
		scope, ok := recipient["scope"].(map[string]any)
		if !ok {
			t.Fatalf("recipients[%d].scope type = %T", i, recipient["scope"])
		}
		version, ok := scope["membership_version"].(json.Number)
		if !ok {
			t.Fatalf("recipients[%d].scope.membership_version type = %T, want number", i, scope["membership_version"])
		}
		gotVersion, err := version.Int64()
		if err != nil || gotVersion != wantVersions[i] {
			t.Fatalf("recipients[%d].scope.membership_version = %q (%v), want %d", i, version, err, wantVersions[i])
		}
	}
}

func TestLegacyNotificationJSONKeepsRealtimeAndPushIDsNumeric(t *testing.T) {
	push := &notificationv1.PushEventPayload{
		UserId:     exactLargeID,
		EventType:  "bid.received",
		Title:      "New bid",
		Body:       "A supplier replied",
		Data:       map[string]string{"request_id": "request-1"},
		Priority:   "high",
		DedupKey:   "bid:1",
		TargetApps: []string{"client"},
		Category:   "promo",
		RecipientScopes: []*notificationv1.NotificationScope{{
			App:               notificationv1.NotificationApp_NOTIFICATION_APP_PRO,
			Perspective:       notificationv1.NotificationPerspective_NOTIFICATION_PERSPECTIVE_SELLER_ORG,
			OrganizationId:    "org-8",
			MembershipVersion: 18,
		}},
	}
	pushWire, err := MarshalLegacyPushEvent(push)
	if err != nil {
		t.Fatalf("marshal legacy push: %v", err)
	}
	assertJSONKeys(t, pushWire, "user_id", "event_type", "title", "body", "data", "priority", "dedup_key", "target_apps", "category")
	assertNumericJSONField(t, pushWire, "user_id", exactLargeID)
	assertNestedMembershipVersion(t, pushWire, "recipient_scopes", 18)

	var pushConsumer struct {
		UserID     int64    `json:"user_id"`
		EventType  string   `json:"event_type"`
		TargetApps []string `json:"target_apps"`
	}
	if err := json.Unmarshal(pushWire, &pushConsumer); err != nil {
		t.Fatalf("current push consumer decode: %v", err)
	}
	if pushConsumer.UserID != exactLargeID || pushConsumer.EventType != "bid.received" ||
		!reflect.DeepEqual(pushConsumer.TargetApps, []string{"client"}) {
		t.Fatalf("push consumer changed legacy values: %+v", pushConsumer)
	}
	var pushRoundTrip notificationv1.PushEventPayload
	if err := UnmarshalLegacyPushEvent(pushWire, &pushRoundTrip); err != nil {
		t.Fatalf("unmarshal legacy push: %v", err)
	}
	if pushRoundTrip.GetUserId() != exactLargeID {
		t.Fatalf("push round-trip user_id = %d", pushRoundTrip.GetUserId())
	}

	createdAt := timestamppb.New(time.Date(2026, time.August, 6, 9, 15, 0, 123000000, time.UTC))
	realtime := &notificationv1.RealtimeNotificationEventPayload{
		UserId:    exactLargeID - 3,
		Id:        "notification-7",
		Type:      "push",
		Category:  "chat",
		Title:     "Message",
		Body:      "New message",
		Data:      map[string]string{"chat_id": "chat-1"},
		CreatedAt: createdAt,
	}
	realtimeWire, err := MarshalLegacyRealtimeNotification(realtime)
	if err != nil {
		t.Fatalf("marshal realtime notification: %v", err)
	}
	assertJSONKeys(t, realtimeWire, "user_id", "id", "type", "category", "title", "body", "data", "is_read", "created_at")
	assertNumericJSONField(t, realtimeWire, "user_id", exactLargeID-3)

	var realtimeConsumer struct {
		UserID    int64     `json:"user_id"`
		ID        string    `json:"id"`
		CreatedAt time.Time `json:"created_at"`
	}
	if err := json.Unmarshal(realtimeWire, &realtimeConsumer); err != nil {
		t.Fatalf("current realtime consumer decode: %v", err)
	}
	if realtimeConsumer.UserID != exactLargeID-3 || realtimeConsumer.ID != "notification-7" ||
		!realtimeConsumer.CreatedAt.Equal(createdAt.AsTime()) {
		t.Fatalf("realtime consumer changed legacy values: %+v", realtimeConsumer)
	}
	var realtimeRoundTrip notificationv1.RealtimeNotificationEventPayload
	if err := UnmarshalLegacyRealtimeNotification(realtimeWire, &realtimeRoundTrip); err != nil {
		t.Fatalf("unmarshal realtime notification: %v", err)
	}
	if realtimeRoundTrip.GetUserId() != exactLargeID-3 || realtimeRoundTrip.GetId() != "notification-7" ||
		!realtimeRoundTrip.GetCreatedAt().AsTime().Equal(createdAt.AsTime()) {
		t.Fatalf("realtime round trip changed values: %+v", &realtimeRoundTrip)
	}
}

func TestProtoJSONIsNotTheLegacyIntegerEnvelope(t *testing.T) {
	event := &chatv1.ChatRealtimeEventPayload{SenderId: 42, RecipientId: 43, RecipientUserIds: []int64{43, 44}}
	typedWire, err := protojson.Marshal(event)
	if err != nil {
		t.Fatalf("marshal typed protojson: %v", err)
	}
	var typedValues map[string]any
	if err := json.Unmarshal(typedWire, &typedValues); err != nil {
		t.Fatalf("decode typed protojson: %v", err)
	}
	if _, ok := typedValues["sender_id"].(string); !ok {
		t.Fatalf("protojson sender_id type = %T, want string", typedValues["sender_id"])
	}
	typedRecipients, ok := typedValues["recipient_user_ids"].([]any)
	if !ok || len(typedRecipients) != 2 {
		t.Fatalf("protojson recipient_user_ids = %#v", typedValues["recipient_user_ids"])
	}
	for i, value := range typedRecipients {
		if _, ok := value.(string); !ok {
			t.Fatalf("protojson recipient_user_ids[%d] type = %T, want string", i, value)
		}
	}
	var legacy chatv1.ChatRealtimeEventPayload
	if err := UnmarshalLegacyChatEvent(typedWire, &legacy); err == nil {
		t.Fatal("legacy encoding/json boundary must reject protojson's quoted int64 envelope")
	}

	legacyWire, err := MarshalLegacyChatEvent(event)
	if err != nil {
		t.Fatalf("marshal legacy chat event: %v", err)
	}
	if bytes.Contains(legacyWire, []byte(`"sender_id":"42"`)) {
		t.Fatalf("legacy boundary emitted quoted int64: %s", legacyWire)
	}
}

func assertNumericJSONField(t *testing.T, data []byte, key string, want int64) {
	t.Helper()
	values := decodeNumbers(t, data)
	number, ok := values[key].(json.Number)
	if !ok {
		t.Fatalf("%s JSON type = %T, want number; wire=%s", key, values[key], data)
	}
	got, err := number.Int64()
	if err != nil || got != want {
		t.Fatalf("%s = %q (%v), want %d", key, number, err, want)
	}
}

func assertNumericJSONArray(t *testing.T, data []byte, key string, want []int64) {
	t.Helper()
	values := decodeNumbers(t, data)
	items, ok := values[key].([]any)
	if !ok || len(items) != len(want) {
		t.Fatalf("%s JSON type/value = %T %#v, want %v", key, values[key], values[key], want)
	}
	for i, item := range items {
		number, ok := item.(json.Number)
		if !ok {
			t.Fatalf("%s[%d] JSON type = %T, want number", key, i, item)
		}
		got, err := number.Int64()
		if err != nil || got != want[i] {
			t.Fatalf("%s[%d] = %q (%v), want %d", key, i, number, err, want[i])
		}
	}
}

func decodeNumbers(t *testing.T, data []byte) map[string]any {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var values map[string]any
	if err := decoder.Decode(&values); err != nil {
		t.Fatalf("decode JSON numbers: %v", err)
	}
	return values
}

func assertJSONKeys(t *testing.T, data []byte, keys ...string) {
	t.Helper()
	values := decodeNumbers(t, data)
	for _, key := range keys {
		if _, ok := values[key]; !ok {
			t.Errorf("legacy JSON missing key %q; wire=%s", key, data)
		}
	}
}

func assertNestedMembershipVersion(t *testing.T, data []byte, scopeKey string, want int64) {
	t.Helper()
	values := decodeNumbers(t, data)
	if _, detached := values["membership_version"]; detached {
		t.Fatalf("membership_version must remain bound inside %s; wire=%s", scopeKey, data)
	}
	scopeValue, ok := values[scopeKey]
	if !ok {
		t.Fatalf("legacy JSON missing canonical scope %q; wire=%s", scopeKey, data)
	}
	if scopes, repeated := scopeValue.([]any); repeated {
		if len(scopes) != 1 {
			t.Fatalf("%s has %d scopes, want 1", scopeKey, len(scopes))
		}
		scopeValue = scopes[0]
	}
	scope, ok := scopeValue.(map[string]any)
	if !ok {
		t.Fatalf("%s JSON type = %T, want object", scopeKey, scopeValue)
	}
	version, ok := scope["membership_version"].(json.Number)
	if !ok {
		t.Fatalf("%s.membership_version JSON type = %T, want number", scopeKey, scope["membership_version"])
	}
	got, err := version.Int64()
	if err != nil || got != want {
		t.Fatalf("%s.membership_version = %q (%v), want %d", scopeKey, version, err, want)
	}
}
