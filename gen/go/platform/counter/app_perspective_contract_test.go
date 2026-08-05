package counterv1

import (
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestCounterAppPerspectiveContract(t *testing.T) {
	enums := File_platform_counter_counter_proto.Enums()
	assertCounterEnumValue(t, enums.ByName("CounterApp"), "COUNTER_APP_CLIENT", 1)
	assertCounterEnumValue(t, enums.ByName("CounterApp"), "COUNTER_APP_PRO", 2)
	assertCounterEnumValue(t, enums.ByName("CounterPerspective"), "COUNTER_PERSPECTIVE_BUYER", 1)
	assertCounterEnumValue(t, enums.ByName("CounterPerspective"), "COUNTER_PERSPECTIVE_SELLER_ORG", 2)
	assertCounterEnumValue(t, enums.ByName("CounterPerspective"), "COUNTER_PERSPECTIVE_SELLER_USER", 3)
	assertCounterEnumValue(t, enums.ByName("CounterPerspective"), "COUNTER_PERSPECTIVE_SUPPORT", 4)

	messages := File_platform_counter_counter_proto.Messages()
	for _, requestName := range []protoreflect.Name{"GetCountersRequest", "GetBadgeTotalRequest"} {
		request := messages.ByName(requestName)
		assertCounterField(t, request, "app", 2, protoreflect.EnumKind, false)
		assertCounterField(t, request, "perspective", 3, protoreflect.EnumKind, false)
		assertCounterField(t, request, "organization_ids", 4, protoreflect.StringKind, true)
	}

	projection := messages.ByName("OrganizationUnreadProjection")
	for _, field := range []counterFieldExpectation{
		{"organization_id", 1, protoreflect.StringKind, false},
		{"unread_messages", 2, protoreflect.Int32Kind, false},
		{"unread_notifications", 3, protoreflect.Int32Kind, false},
		{"total", 4, protoreflect.Int32Kind, false},
	} {
		assertCounterField(t, projection, field.name, field.number, field.kind, field.list)
	}
	assertCounterField(t, messages.ByName("GetCountersResponse"), "organization_unread", 2, protoreflect.MessageKind, true)
	assertCounterField(t, messages.ByName("GetBadgeTotalResponse"), "organization_unread", 3, protoreflect.MessageKind, true)
}

type counterFieldExpectation struct {
	name   protoreflect.Name
	number protoreflect.FieldNumber
	kind   protoreflect.Kind
	list   bool
}

func assertCounterEnumValue(t *testing.T, enum protoreflect.EnumDescriptor, name protoreflect.Name, number protoreflect.EnumNumber) {
	t.Helper()
	if enum == nil {
		t.Fatalf("enum containing %s not found", name)
	}
	value := enum.Values().ByName(name)
	if value == nil || value.Number() != number {
		t.Fatalf("enum value %s = %v, want number %d", name, value, number)
	}
}

func assertCounterField(t *testing.T, message protoreflect.MessageDescriptor, name protoreflect.Name, number protoreflect.FieldNumber, kind protoreflect.Kind, list bool) {
	t.Helper()
	if message == nil {
		t.Fatalf("message containing %s not found", name)
	}
	field := message.Fields().ByName(name)
	if field == nil {
		t.Fatalf("%s.%s not found", message.Name(), name)
	}
	if field.Number() != number || field.Kind() != kind || field.IsList() != list {
		t.Fatalf("%s.%s = field %d (%s, list=%t), want %d (%s, list=%t)", message.Name(), name, field.Number(), field.Kind(), field.IsList(), number, kind, list)
	}
}
