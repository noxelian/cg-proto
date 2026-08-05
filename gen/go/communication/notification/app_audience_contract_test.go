package notificationv1

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestAppAudienceContract(t *testing.T) {
	enums := File_communication_notification_notification_proto.Enums()
	assertNotificationEnumValue(t, enums.ByName("NotificationApp"), "NOTIFICATION_APP_CLIENT", 1)
	assertNotificationEnumValue(t, enums.ByName("NotificationApp"), "NOTIFICATION_APP_PRO", 2)
	assertNotificationEnumValue(t, enums.ByName("NotificationPerspective"), "NOTIFICATION_PERSPECTIVE_BUYER", 1)
	assertNotificationEnumValue(t, enums.ByName("NotificationPerspective"), "NOTIFICATION_PERSPECTIVE_SELLER_ORG", 2)
	assertNotificationEnumValue(t, enums.ByName("NotificationPerspective"), "NOTIFICATION_PERSPECTIVE_SELLER_USER", 3)
	assertNotificationEnumValue(t, enums.ByName("NotificationPerspective"), "NOTIFICATION_PERSPECTIVE_SUPPORT", 4)
	assertNotificationEnumValue(t, enums.ByName("NotificationPerspective"), "NOTIFICATION_PERSPECTIVE_BUYER_ORG", 5)

	messages := File_communication_notification_notification_proto.Messages()
	for _, field := range []notificationFieldExpectation{
		{"recipient_app", 10, protoreflect.EnumKind},
		{"recipient_perspective", 11, protoreflect.EnumKind},
		{"recipient_organization_id", 12, protoreflect.StringKind},
		{"context_type", 13, protoreflect.StringKind},
		{"context_id", 14, protoreflect.StringKind},
	} {
		assertNotificationField(t, messages.ByName("Notification"), field.name, field.number, field.kind)
	}

	requests := map[protoreflect.Name][]notificationFieldExpectation{
		"SendPushRequest":               {{"recipient_app", 6, protoreflect.EnumKind}, {"recipient_perspective", 7, protoreflect.EnumKind}, {"recipient_organization_id", 8, protoreflect.StringKind}, {"context_type", 9, protoreflect.StringKind}, {"context_id", 10, protoreflect.StringKind}},
		"GetNotificationsRequest":       {{"app", 7, protoreflect.EnumKind}, {"perspective", 8, protoreflect.EnumKind}, {"organization_id", 9, protoreflect.StringKind}},
		"GetNotificationStreamsRequest": {{"app", 2, protoreflect.EnumKind}, {"perspective", 3, protoreflect.EnumKind}, {"organization_id", 4, protoreflect.StringKind}},
		"MarkAsReadRequest":             {{"app", 5, protoreflect.EnumKind}, {"perspective", 6, protoreflect.EnumKind}, {"organization_id", 7, protoreflect.StringKind}},
		"GetUnreadCountRequest":         {{"app", 2, protoreflect.EnumKind}, {"perspective", 3, protoreflect.EnumKind}, {"organization_id", 4, protoreflect.StringKind}},
	}
	for messageName, fields := range requests {
		for _, field := range fields {
			assertNotificationField(t, messages.ByName(messageName), field.name, field.number, field.kind)
		}
	}

	pushEvent := messages.ByName("PushEventPayload")
	for _, field := range []notificationFieldExpectation{
		{"target_apps", 8, protoreflect.StringKind},
		{"category", 9, protoreflect.StringKind},
		{"typed_target_apps", 10, protoreflect.EnumKind},
		{"recipient_perspective", 11, protoreflect.EnumKind},
		{"recipient_organization_id", 12, protoreflect.StringKind},
		{"context_type", 13, protoreflect.StringKind},
		{"context_id", 14, protoreflect.StringKind},
		{"typed_category", 15, protoreflect.EnumKind},
	} {
		assertNotificationField(t, pushEvent, field.name, field.number, field.kind)
	}
	for _, fieldName := range []protoreflect.Name{"target_apps", "typed_target_apps"} {
		if !pushEvent.Fields().ByName(fieldName).IsList() {
			t.Fatalf("PushEventPayload.%s must be repeated", fieldName)
		}
	}

	realtimeEvent := messages.ByName("RealtimeNotificationEventPayload")
	for _, field := range []notificationFieldExpectation{
		{"type", 3, protoreflect.StringKind},
		{"category", 4, protoreflect.StringKind},
		{"recipient_app", 10, protoreflect.EnumKind},
		{"recipient_perspective", 11, protoreflect.EnumKind},
		{"recipient_organization_id", 12, protoreflect.StringKind},
		{"context_type", 13, protoreflect.StringKind},
		{"context_id", 14, protoreflect.StringKind},
		{"typed_type", 15, protoreflect.EnumKind},
		{"typed_category", 16, protoreflect.EnumKind},
	} {
		assertNotificationField(t, realtimeEvent, field.name, field.number, field.kind)
	}

	registerDevice := messages.ByName("RegisterDeviceRequest")
	assertNotificationField(t, registerDevice, "app", 8, protoreflect.StringKind)
	assertNotificationField(t, registerDevice, "target_app", 9, protoreflect.EnumKind)
	assertNotificationContractDocumentation(t)
}

func TestPushEventPayloadProtoJSONPreservesKeysButQuotesInt64(t *testing.T) {
	payload := &PushEventPayload{
		UserId:               42,
		EventType:            "bid.received",
		Title:                "New bid",
		Body:                 "A workshop replied",
		Data:                 map[string]string{"route": "request"},
		Priority:             "high",
		DedupKey:             "bid:77",
		TargetApps:           []string{"client", "partner"},
		Category:             "promo",
		TypedTargetApps:      []NotificationApp{NotificationApp_NOTIFICATION_APP_CLIENT, NotificationApp_NOTIFICATION_APP_PRO},
		RecipientPerspective: NotificationPerspective_NOTIFICATION_PERSPECTIVE_BUYER,
		TypedCategory:        NotificationCategory_NOTIFICATION_CATEGORY_PROMO,
	}

	wire := marshalNotificationProtoJSON(t, payload)
	assertNotificationJSONKeys(t, wire,
		[]string{"user_id", "event_type", "dedup_key", "target_apps", "category"},
		[]string{"userId", "eventType", "dedupKey", "targetApps"},
	)
	if got := string(wire["user_id"]); got != `"42"` {
		t.Fatalf("protojson user_id = %s, want quoted int64; legacy events must use compat/communication", got)
	}
	if _, ok := wire["typedTargetApps"]; !ok {
		t.Fatal("deprecated typedTargetApps must remain wire-readable during migration")
	}

	var consumer struct {
		EventType  string   `json:"event_type"`
		DedupKey   string   `json:"dedup_key"`
		TargetApps []string `json:"target_apps"`
		Category   string   `json:"category"`
	}
	decodeNotificationJSON(t, wire, &consumer)
	if consumer.EventType != "bid.received" || consumer.DedupKey != "bid:77" || consumer.Category != "promo" {
		t.Fatalf("encoding/json migration seam lost legacy values: %+v", consumer)
	}
	if want := []string{"client", "partner"}; !reflect.DeepEqual(consumer.TargetApps, want) {
		t.Fatalf("encoding/json target_apps = %v, want %v", consumer.TargetApps, want)
	}
}

func TestPushEventPayloadProtoJSONReadsLiveLegacyJSONWithoutDefaulting(t *testing.T) {
	const live = `{"user_id":42,"event_type":"bid.received","title":"New bid","body":"A workshop replied","data":{"route":"request"},"priority":"high","dedup_key":"bid:77","target_apps":["client","partner"],"category":"promo"}`
	var payload PushEventPayload
	if err := protojson.Unmarshal([]byte(live), &payload); err != nil {
		t.Fatalf("unmarshal live notification.push JSON: %v", err)
	}
	if payload.UserId != 42 || payload.EventType != "bid.received" || payload.DedupKey != "bid:77" || payload.Category != "promo" {
		t.Fatalf("legacy push values were lost or defaulted: %+v", &payload)
	}
	if want := []string{"client", "partner"}; !reflect.DeepEqual(payload.TargetApps, want) {
		t.Fatalf("target_apps = %v, want %v", payload.TargetApps, want)
	}
}

func TestRealtimeNotificationEventProtoJSONPreservesKeysButQuotesInt64(t *testing.T) {
	createdAt := timestamppb.New(time.Date(2026, time.August, 5, 12, 30, 0, 0, time.UTC))
	payload := &RealtimeNotificationEventPayload{
		UserId:        42,
		Id:            "notification-7",
		Type:          "push",
		Category:      "chat",
		Title:         "Message",
		Body:          "New message",
		Data:          map[string]string{"chat_id": "chat-1"},
		IsRead:        true,
		CreatedAt:     createdAt,
		TypedType:     NotificationType_NOTIFICATION_TYPE_PUSH,
		TypedCategory: NotificationCategory_NOTIFICATION_CATEGORY_CHAT,
	}

	wire := marshalNotificationProtoJSON(t, payload)
	assertNotificationJSONKeys(t, wire,
		[]string{"user_id", "type", "category", "is_read", "created_at"},
		[]string{"userId", "isRead", "createdAt"},
	)
	if got := string(wire["user_id"]); got != `"42"` {
		t.Fatalf("protojson user_id = %s, want quoted int64; legacy events must use compat/communication", got)
	}
	if _, ok := wire["typedType"]; !ok {
		t.Fatal("typedType must coexist separately from legacy string type")
	}
	if _, ok := wire["typedCategory"]; !ok {
		t.Fatal("typedCategory must coexist separately from legacy string category")
	}

	var consumer struct {
		Type      string    `json:"type"`
		Category  string    `json:"category"`
		IsRead    bool      `json:"is_read"`
		CreatedAt time.Time `json:"created_at"`
	}
	decodeNotificationJSON(t, wire, &consumer)
	if consumer.Type != "push" || consumer.Category != "chat" || !consumer.IsRead ||
		consumer.CreatedAt.UTC().Format(time.RFC3339) != "2026-08-05T12:30:00Z" {
		t.Fatalf("encoding/json realtime seam lost legacy values: %+v", consumer)
	}
}

func TestRealtimeNotificationEventProtoJSONReadsLiveLegacyJSONWithoutDefaulting(t *testing.T) {
	const live = `{"user_id":42,"id":"notification-7","type":"push","category":"chat","title":"Message","body":"New message","data":{"chat_id":"chat-1"},"is_read":true,"created_at":"2026-08-05T12:30:00Z"}`
	var payload RealtimeNotificationEventPayload
	if err := protojson.Unmarshal([]byte(live), &payload); err != nil {
		t.Fatalf("unmarshal live notification.new JSON: %v", err)
	}
	if payload.UserId != 42 || payload.Type != "push" || payload.Category != "chat" || !payload.IsRead {
		t.Fatalf("legacy realtime values were lost or defaulted: %+v", &payload)
	}
	if got := payload.CreatedAt.AsTime().UTC().Format(time.RFC3339); got != "2026-08-05T12:30:00Z" {
		t.Fatalf("created_at = %q, want live value", got)
	}
}

func marshalNotificationProtoJSON(t *testing.T, message interface{ ProtoReflect() protoreflect.Message }) map[string]json.RawMessage {
	t.Helper()
	encoded, err := protojson.Marshal(message)
	if err != nil {
		t.Fatalf("protojson marshal: %v", err)
	}
	var wire map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &wire); err != nil {
		t.Fatalf("decode marshaled protojson: %v", err)
	}
	return wire
}

func assertNotificationJSONKeys(t *testing.T, wire map[string]json.RawMessage, present, absent []string) {
	t.Helper()
	for _, key := range present {
		if _, ok := wire[key]; !ok {
			t.Errorf("protojson missing required legacy key %q; keys=%v", key, notificationJSONKeys(wire))
		}
	}
	for _, key := range absent {
		if _, ok := wire[key]; ok {
			t.Errorf("protojson emitted forbidden camelCase legacy key %q", key)
		}
	}
}

func notificationJSONKeys(wire map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(wire))
	for key := range wire {
		keys = append(keys, key)
	}
	return keys
}

func decodeNotificationJSON(t *testing.T, wire map[string]json.RawMessage, target any) {
	t.Helper()
	encoded, err := json.Marshal(wire)
	if err != nil {
		t.Fatalf("encode protojson map: %v", err)
	}
	if err := json.Unmarshal(encoded, target); err != nil {
		t.Fatalf("decode with current encoding/json tags: %v", err)
	}
}

func assertNotificationContractDocumentation(t *testing.T) {
	t.Helper()
	source, err := os.ReadFile("../../../../communication/notification/notification.proto")
	if err != nil {
		t.Fatalf("read notification.proto: %v", err)
	}
	for _, required := range []string{
		`("client" -> CLIENT, "partner" -> PRO)`,
		"owner rejects INVALID_ARGUMENT",
		"broadcast-to-",
		"MUST NOT become CLIENT-only or drop",
		"empty-to-CLIENT fallback is legacy-ingress-only",
		"MUST match the verified JWT",
		"BFF-selected",
		"fields are never authority",
	} {
		if !strings.Contains(string(source), required) {
			t.Errorf("notification.proto missing contract documentation %q", required)
		}
	}
}

type notificationFieldExpectation struct {
	name   protoreflect.Name
	number protoreflect.FieldNumber
	kind   protoreflect.Kind
}

func assertNotificationEnumValue(t *testing.T, enum protoreflect.EnumDescriptor, name protoreflect.Name, number protoreflect.EnumNumber) {
	t.Helper()
	if enum == nil {
		t.Fatalf("enum containing %s not found", name)
	}
	value := enum.Values().ByName(name)
	if value == nil || value.Number() != number {
		t.Fatalf("enum value %s = %v, want number %d", name, value, number)
	}
}

func assertNotificationField(t *testing.T, message protoreflect.MessageDescriptor, name protoreflect.Name, number protoreflect.FieldNumber, kind protoreflect.Kind) protoreflect.FieldDescriptor {
	t.Helper()
	if message == nil {
		t.Fatalf("message containing %s not found", name)
	}
	field := message.Fields().ByName(name)
	if field == nil {
		t.Fatalf("%s.%s not found", message.Name(), name)
	}
	if field.Number() != number || field.Kind() != kind {
		t.Fatalf("%s.%s = field %d (%s), want %d (%s)", message.Name(), name, field.Number(), field.Kind(), number, kind)
	}
	return field
}
