package chatv1

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestAppPerspectiveContract(t *testing.T) {
	enums := File_communication_chat_chat_proto.Enums()
	assertEnumValue(t, enums.ByName("ChatContextType"), "CHAT_CONTEXT_TYPE_REQUEST", 2)
	assertEnumValue(t, enums.ByName("ChatPerspective"), "CHAT_PERSPECTIVE_BUYER", 1)
	assertEnumValue(t, enums.ByName("ChatPerspective"), "CHAT_PERSPECTIVE_SELLER_ORG", 2)
	assertEnumValue(t, enums.ByName("ChatPerspective"), "CHAT_PERSPECTIVE_SELLER_USER", 3)
	assertEnumValue(t, enums.ByName("ChatPerspective"), "CHAT_PERSPECTIVE_SUPPORT", 4)
	assertEnumValue(t, enums.ByName("ChatApp"), "CHAT_APP_CLIENT", 1)
	assertEnumValue(t, enums.ByName("ChatApp"), "CHAT_APP_PRO", 2)

	messages := File_communication_chat_chat_proto.Messages()
	assertField(t, messages.ByName("Chat"), "caller_perspective", 20, protoreflect.EnumKind)
	assertField(t, messages.ByName("Chat"), "caller_organization_id", 21, protoreflect.StringKind)
	assertField(t, messages.ByName("Chat"), "vehicle_context", 22, protoreflect.MessageKind)
	assertField(t, messages.ByName("Chat"), "origin_bid_id", 23, protoreflect.Int64Kind)
	assertField(t, messages.ByName("Message"), "sender_perspective", 8, protoreflect.EnumKind)
	assertField(t, messages.ByName("Message"), "sender_organization_id", 9, protoreflect.StringKind)

	requestFields := map[protoreflect.Name][]fieldExpectation{
		"CreateChatRequest":    {{"perspective", 6, protoreflect.EnumKind}, {"organization_id", 7, protoreflect.StringKind}, {"origin_bid_id", 8, protoreflect.Int64Kind}, {"app", 9, protoreflect.EnumKind}},
		"GetChatRequest":       {{"perspective", 2, protoreflect.EnumKind}, {"organization_id", 3, protoreflect.StringKind}, {"app", 4, protoreflect.EnumKind}},
		"GetUserChatsRequest":  {{"perspective", 4, protoreflect.EnumKind}, {"organization_id", 5, protoreflect.StringKind}, {"app", 6, protoreflect.EnumKind}},
		"GetOrgChatsRequest":   {{"organization_id", 1, protoreflect.StringKind}, {"perspective", 5, protoreflect.EnumKind}, {"app", 6, protoreflect.EnumKind}},
		"SendMessageRequest":   {{"perspective", 5, protoreflect.EnumKind}, {"organization_id", 6, protoreflect.StringKind}, {"app", 7, protoreflect.EnumKind}},
		"GetMessagesRequest":   {{"perspective", 4, protoreflect.EnumKind}, {"organization_id", 5, protoreflect.StringKind}, {"app", 6, protoreflect.EnumKind}},
		"MarkAsReadRequest":    {{"perspective", 4, protoreflect.EnumKind}, {"organization_id", 5, protoreflect.StringKind}, {"app", 6, protoreflect.EnumKind}},
		"DeleteMessageRequest": {{"perspective", 4, protoreflect.EnumKind}, {"organization_id", 5, protoreflect.StringKind}, {"app", 6, protoreflect.EnumKind}},
	}
	for messageName, fields := range requestFields {
		message := messages.ByName(messageName)
		for _, field := range fields {
			assertField(t, message, field.name, field.number, field.kind)
		}
	}

	vehicle := messages.ByName("VehicleContext")
	for _, field := range []fieldExpectation{
		{"garage_car_id", 1, protoreflect.Int64Kind},
		{"car_make_id", 2, protoreflect.Int64Kind},
		{"car_model_id", 3, protoreflect.Int64Kind},
		{"car_generation_id", 4, protoreflect.Int64Kind},
		{"year", 5, protoreflect.Int32Kind},
		{"vin", 6, protoreflect.StringKind},
		{"provenance", 7, protoreflect.EnumKind},
	} {
		assertField(t, vehicle, field.name, field.number, field.kind)
	}
	if vehicle != nil && vehicle.Fields().ByName("logo_url") != nil {
		t.Fatal("VehicleContext must not carry NSI logo_url")
	}
	assertField(t, messages.ByName("GetMessagesResponse"), "vehicle_context", 3, protoreflect.MessageKind)

	realtime := messages.ByName("ChatRealtimeEventPayload")
	for _, field := range []fieldExpectation{
		{"target_apps", 7, protoreflect.StringKind},
		{"recipient_org_id", 8, protoreflect.StringKind},
		{"recipient_app", 9, protoreflect.EnumKind},
		{"recipient_perspective", 10, protoreflect.EnumKind},
		{"recipient_organization_id", 11, protoreflect.StringKind},
		{"context_type", 12, protoreflect.StringKind},
		{"context_id", 13, protoreflect.StringKind},
	} {
		assertField(t, realtime, field.name, field.number, field.kind)
	}
	if !realtime.Fields().ByName("target_apps").IsList() {
		t.Fatal("ChatRealtimeEventPayload.target_apps must remain a repeated legacy JSON string")
	}

	assertChatContractDocumentation(t)
}

func TestChatRealtimeEventProtoJSONPreservesLegacyWireShape(t *testing.T) {
	payload := &ChatRealtimeEventPayload{
		MessageId:               "message-1",
		ChatId:                  "chat-1",
		SenderId:                11,
		MessageType:             "text",
		RecipientId:             22,
		RecipientUserIds:        []int64{22, 23},
		TargetApps:              []string{"client", "partner"},
		RecipientOrgId:          "org-legacy",
		RecipientApp:            ChatApp_CHAT_APP_PRO,
		RecipientPerspective:    ChatPerspective_CHAT_PERSPECTIVE_SELLER_ORG,
		RecipientOrganizationId: "org-typed",
		ContextType:             "request",
		ContextId:               "request-7",
	}

	encoded, err := protojson.Marshal(payload)
	if err != nil {
		t.Fatalf("protojson marshal chat event: %v", err)
	}
	var wire map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &wire); err != nil {
		t.Fatalf("decode marshaled chat protojson: %v", err)
	}
	for _, key := range []string{"message_id", "chat_id", "sender_id", "message_type", "recipient_id", "recipient_user_ids", "target_apps", "recipient_org_id", "context_type", "context_id"} {
		if _, ok := wire[key]; !ok {
			t.Errorf("protojson missing required legacy key %q", key)
		}
	}
	for _, key := range []string{"messageId", "chatId", "senderId", "messageType", "recipientId", "recipientUserIds", "targetApps", "recipientOrgId", "contextType", "contextId"} {
		if _, ok := wire[key]; ok {
			t.Errorf("protojson emitted forbidden camelCase legacy key %q", key)
		}
	}
	if _, ok := wire["recipientApp"]; !ok {
		t.Fatal("typed recipientApp must coexist separately from legacy target_apps")
	}
	if _, ok := wire["recipientOrganizationId"]; !ok {
		t.Fatal("typed recipientOrganizationId must coexist separately from legacy recipient_org_id")
	}

	var consumer struct {
		MessageID      string   `json:"message_id"`
		ChatID         string   `json:"chat_id"`
		MessageType    string   `json:"message_type"`
		TargetApps     []string `json:"target_apps"`
		RecipientOrgID string   `json:"recipient_org_id"`
		ContextType    string   `json:"context_type"`
		ContextID      string   `json:"context_id"`
	}
	if err := json.Unmarshal(encoded, &consumer); err != nil {
		t.Fatalf("decode chat protojson with current encoding/json tags: %v", err)
	}
	if consumer.MessageID != "message-1" || consumer.ChatID != "chat-1" || consumer.MessageType != "text" ||
		consumer.RecipientOrgID != "org-legacy" || consumer.ContextType != "request" || consumer.ContextID != "request-7" {
		t.Fatalf("encoding/json chat migration seam lost legacy values: %+v", consumer)
	}
	if want := []string{"client", "partner"}; !reflect.DeepEqual(consumer.TargetApps, want) {
		t.Fatalf("encoding/json target_apps = %v, want %v", consumer.TargetApps, want)
	}
}

func TestChatRealtimeEventProtoJSONReadsLiveLegacyJSONWithoutDefaulting(t *testing.T) {
	const live = `{"message_id":"message-1","chat_id":"chat-1","sender_id":11,"message_type":"text","recipient_id":22,"recipient_user_ids":[22,23],"target_apps":["client","partner"],"recipient_org_id":"org-legacy","context_type":"request","context_id":"request-7"}`
	var payload ChatRealtimeEventPayload
	if err := protojson.Unmarshal([]byte(live), &payload); err != nil {
		t.Fatalf("unmarshal live chat.message.sent JSON: %v", err)
	}
	if payload.MessageId != "message-1" || payload.ChatId != "chat-1" || payload.SenderId != 11 || payload.MessageType != "text" ||
		payload.RecipientId != 22 || payload.RecipientOrgId != "org-legacy" || payload.ContextType != "request" || payload.ContextId != "request-7" {
		t.Fatalf("legacy chat values were lost or defaulted: %+v", &payload)
	}
	if !reflect.DeepEqual(payload.RecipientUserIds, []int64{22, 23}) {
		t.Fatalf("recipient_user_ids = %v, want [22 23]", payload.RecipientUserIds)
	}
	if !reflect.DeepEqual(payload.TargetApps, []string{"client", "partner"}) {
		t.Fatalf("target_apps = %v, want [client partner]", payload.TargetApps)
	}
}

func TestChatRealtimeEventKeepsConflictingFormsForOwnerValidation(t *testing.T) {
	const conflicting = `{"target_apps":["client"],"recipient_org_id":"org-legacy","recipientApp":"CHAT_APP_PRO","recipientOrganizationId":"org-typed"}`
	var payload ChatRealtimeEventPayload
	if err := protojson.Unmarshal([]byte(conflicting), &payload); err != nil {
		t.Fatalf("unmarshal conflicting chat dual-write fixture: %v", err)
	}
	if !reflect.DeepEqual(payload.TargetApps, []string{"client"}) || payload.RecipientOrgId != "org-legacy" ||
		payload.RecipientApp != ChatApp_CHAT_APP_PRO || payload.RecipientOrganizationId != "org-typed" {
		t.Fatalf("legacy and typed forms must remain distinct for owner rejection: %+v", &payload)
	}
}

func assertChatContractDocumentation(t *testing.T) {
	t.Helper()
	source, err := os.ReadFile("../../../../communication/chat/chat.proto")
	if err != nil {
		t.Fatalf("read chat.proto: %v", err)
	}
	for _, required := range []string{
		`("client" -> CLIENT, "partner" -> PRO)`,
		"they MUST agree or the owner rejects INVALID_ARGUMENT",
		"MUST retain legacy fallback delivery",
		"owner/BFF MUST bind app to the verified JWT app",
		"request values are never authority",
		"CLIENT+BUYER and PRO+SELLER_ORG",
	} {
		if !strings.Contains(string(source), required) {
			t.Errorf("chat.proto missing contract documentation %q", required)
		}
	}
}

type fieldExpectation struct {
	name   protoreflect.Name
	number protoreflect.FieldNumber
	kind   protoreflect.Kind
}

func assertEnumValue(t *testing.T, enum protoreflect.EnumDescriptor, name protoreflect.Name, number protoreflect.EnumNumber) {
	t.Helper()
	if enum == nil {
		t.Fatalf("enum containing %s not found", name)
	}
	value := enum.Values().ByName(name)
	if value == nil {
		t.Fatalf("enum value %s not found", name)
	}
	if value.Number() != number {
		t.Fatalf("%s number = %d, want %d", name, value.Number(), number)
	}
}

func assertField(t *testing.T, message protoreflect.MessageDescriptor, name protoreflect.Name, number protoreflect.FieldNumber, kind protoreflect.Kind) {
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
}
