package chatv1

import (
	"testing"

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
		"GetChatRequest":      {{"perspective", 2, protoreflect.EnumKind}, {"organization_id", 3, protoreflect.StringKind}},
		"GetUserChatsRequest": {{"perspective", 4, protoreflect.EnumKind}, {"organization_id", 5, protoreflect.StringKind}},
		"GetOrgChatsRequest":  {{"perspective", 5, protoreflect.EnumKind}},
		"SendMessageRequest":  {{"perspective", 5, protoreflect.EnumKind}, {"organization_id", 6, protoreflect.StringKind}},
		"GetMessagesRequest":  {{"perspective", 4, protoreflect.EnumKind}, {"organization_id", 5, protoreflect.StringKind}},
		"MarkAsReadRequest":   {{"perspective", 4, protoreflect.EnumKind}, {"organization_id", 5, protoreflect.StringKind}},
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
		{"recipient_app", 7, protoreflect.EnumKind},
		{"recipient_perspective", 8, protoreflect.EnumKind},
		{"recipient_organization_id", 9, protoreflect.StringKind},
		{"context_type", 10, protoreflect.EnumKind},
		{"context_id", 11, protoreflect.StringKind},
	} {
		assertField(t, realtime, field.name, field.number, field.kind)
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
