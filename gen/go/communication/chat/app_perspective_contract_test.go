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
	assertEnumValue(t, enums.ByName("ChatApp"), "CHAT_APP_ADMIN", 3)

	messages := File_communication_chat_chat_proto.Messages()
	chatScope := messages.ByName("ChatScope")
	assertField(t, chatScope, "app", 1, protoreflect.EnumKind)
	assertField(t, chatScope, "perspective", 2, protoreflect.EnumKind)
	assertField(t, chatScope, "organization_id", 3, protoreflect.StringKind)
	assertField(t, chatScope, "membership_version", 4, protoreflect.Int64Kind)
	assertField(t, messages.ByName("Chat"), "caller_perspective", 20, protoreflect.EnumKind)
	assertField(t, messages.ByName("Chat"), "caller_organization_id", 21, protoreflect.StringKind)
	assertField(t, messages.ByName("Chat"), "vehicle_context", 22, protoreflect.MessageKind)
	assertField(t, messages.ByName("Chat"), "origin_bid_id", 23, protoreflect.Int64Kind)
	assertField(t, messages.ByName("Chat"), "caller_scope", 24, protoreflect.MessageKind)
	assertField(t, messages.ByName("Message"), "sender_perspective", 8, protoreflect.EnumKind)
	assertField(t, messages.ByName("Message"), "sender_organization_id", 9, protoreflect.StringKind)
	assertField(t, messages.ByName("Message"), "sender_scope", 10, protoreflect.MessageKind)

	requestFields := map[protoreflect.Name][]fieldExpectation{
		"CreateChatRequest":    {{"perspective", 6, protoreflect.EnumKind}, {"organization_id", 7, protoreflect.StringKind}, {"origin_bid_id", 8, protoreflect.Int64Kind}, {"app", 9, protoreflect.EnumKind}, {"scope", 10, protoreflect.MessageKind}},
		"GetChatRequest":       {{"perspective", 2, protoreflect.EnumKind}, {"organization_id", 3, protoreflect.StringKind}, {"app", 4, protoreflect.EnumKind}, {"scope", 5, protoreflect.MessageKind}},
		"GetUserChatsRequest":  {{"perspective", 4, protoreflect.EnumKind}, {"organization_id", 5, protoreflect.StringKind}, {"app", 6, protoreflect.EnumKind}, {"scope", 7, protoreflect.MessageKind}},
		"GetOrgChatsRequest":   {{"organization_id", 1, protoreflect.StringKind}, {"perspective", 5, protoreflect.EnumKind}, {"app", 6, protoreflect.EnumKind}, {"scope", 7, protoreflect.MessageKind}},
		"SendMessageRequest":   {{"perspective", 5, protoreflect.EnumKind}, {"organization_id", 6, protoreflect.StringKind}, {"app", 7, protoreflect.EnumKind}, {"scope", 8, protoreflect.MessageKind}},
		"GetMessagesRequest":   {{"perspective", 4, protoreflect.EnumKind}, {"organization_id", 5, protoreflect.StringKind}, {"app", 6, protoreflect.EnumKind}, {"scope", 7, protoreflect.MessageKind}},
		"MarkAsReadRequest":    {{"perspective", 4, protoreflect.EnumKind}, {"organization_id", 5, protoreflect.StringKind}, {"app", 6, protoreflect.EnumKind}, {"scope", 7, protoreflect.MessageKind}},
		"DeleteMessageRequest": {{"perspective", 4, protoreflect.EnumKind}, {"organization_id", 5, protoreflect.StringKind}, {"app", 6, protoreflect.EnumKind}, {"scope", 7, protoreflect.MessageKind}},
	}
	for messageName, fields := range requestFields {
		message := messages.ByName(messageName)
		for _, field := range fields {
			assertField(t, message, field.name, field.number, field.kind)
		}
	}
	chatService := File_communication_chat_chat_proto.Services().ByName("ChatService")
	if chatService == nil {
		t.Fatal("ChatService not found")
	}
	if chatService.Methods().Len() != len(requestFields) {
		t.Fatalf("ChatService has %d methods, test enumerates %d; every adjacent chat RPC must carry scope", chatService.Methods().Len(), len(requestFields))
	}
	for i := 0; i < chatService.Methods().Len(); i++ {
		method := chatService.Methods().Get(i)
		if _, ok := requestFields[method.Input().Name()]; !ok {
			t.Fatalf("chat RPC %s request %s is not enumerated", method.Name(), method.Input().Name())
		}
		assertField(t, method.Input(), "scope", method.Input().Fields().ByName("scope").Number(), protoreflect.MessageKind)
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
	assertField(t, realtime, "recipient_scope", 14, protoreflect.MessageKind)

	adminRequestFields := map[protoreflect.Name]protoreflect.FieldNumber{
		"AdminGetUserChatsRequest":    4,
		"AdminGetChatMessagesRequest": 4,
		"AdminListChatsRequest":       4,
		"AdminSendMessageRequest":     3,
		"AdminMarkChatReadRequest":    2,
	}
	adminService := File_communication_chat_chat_proto.Services().ByName("AdminChatService")
	if adminService == nil {
		t.Fatal("AdminChatService not found")
	}
	if adminService.Methods().Len() != len(adminRequestFields) {
		t.Fatalf("AdminChatService has %d methods, test enumerates %d; add every adjacent support method to the scope invariant", adminService.Methods().Len(), len(adminRequestFields))
	}
	for i := 0; i < adminService.Methods().Len(); i++ {
		method := adminService.Methods().Get(i)
		fieldNumber, ok := adminRequestFields[method.Input().Name()]
		if !ok {
			t.Fatalf("admin chat RPC %s request %s is not enumerated", method.Name(), method.Input().Name())
		}
		assertField(t, method.Input(), "scope", fieldNumber, protoreflect.MessageKind)
	}

	assertChatContractDocumentation(t)
}

func TestChatRealtimeEventProtoJSONPreservesKeysButQuotesInt64(t *testing.T) {
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
	for key, want := range map[string]string{
		"sender_id":    `"11"`,
		"recipient_id": `"22"`,
	} {
		if got := string(wire[key]); got != want {
			t.Fatalf("protojson %s = %s, want %s; legacy events must use compat/communication", key, got, want)
		}
	}
	var protoJSONRecipientIDs []string
	if err := json.Unmarshal(wire["recipient_user_ids"], &protoJSONRecipientIDs); err != nil ||
		!reflect.DeepEqual(protoJSONRecipientIDs, []string{"22", "23"}) {
		t.Fatalf("protojson recipient_user_ids = %s (%v), want quoted strings; legacy events must use compat/communication", wire["recipient_user_ids"], err)
	}
	if _, ok := wire["recipientApp"]; !ok {
		t.Fatal("deprecated recipientApp must remain wire-readable during migration")
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
			"membership_version is 0 iff organization_id is empty",
			"stale mismatch",
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
