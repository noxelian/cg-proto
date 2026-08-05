package counterv1

import (
	"os"
	"strings"
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
	scope := messages.ByName("CounterScope")
	assertCounterField(t, scope, "app", 1, protoreflect.EnumKind, false)
	assertCounterField(t, scope, "perspective", 2, protoreflect.EnumKind, false)
	assertCounterField(t, scope, "organization_id", 3, protoreflect.StringKind, false)
	assertCounterField(t, scope, "membership_version", 4, protoreflect.Int64Kind, false)
	requests := map[protoreflect.Name][]counterFieldExpectation{
		"GetCountersRequest":                  {{"app", 2, protoreflect.EnumKind, false}, {"perspective", 3, protoreflect.EnumKind, false}, {"organization_ids", 4, protoreflect.StringKind, true}, {"scopes", 5, protoreflect.MessageKind, true}},
		"IncrementCounterRequest":             {{"app", 4, protoreflect.EnumKind, false}, {"perspective", 5, protoreflect.EnumKind, false}, {"organization_ids", 6, protoreflect.StringKind, true}, {"scopes", 7, protoreflect.MessageKind, true}},
		"DecrementCounterRequest":             {{"app", 4, protoreflect.EnumKind, false}, {"perspective", 5, protoreflect.EnumKind, false}, {"organization_ids", 6, protoreflect.StringKind, true}, {"scopes", 7, protoreflect.MessageKind, true}},
		"SetCounterRequest":                   {{"app", 4, protoreflect.EnumKind, false}, {"perspective", 5, protoreflect.EnumKind, false}, {"organization_ids", 6, protoreflect.StringKind, true}, {"scopes", 7, protoreflect.MessageKind, true}},
		"GetRoadsidePurchasesSnapshotRequest": {{"app", 2, protoreflect.EnumKind, false}, {"perspective", 3, protoreflect.EnumKind, false}, {"organization_ids", 4, protoreflect.StringKind, true}, {"scopes", 5, protoreflect.MessageKind, true}},
		"ResetRoadsidePurchasesUnreadRequest": {{"app", 3, protoreflect.EnumKind, false}, {"perspective", 4, protoreflect.EnumKind, false}, {"organization_ids", 5, protoreflect.StringKind, true}, {"scopes", 6, protoreflect.MessageKind, true}},
		"GetBadgeTotalRequest":                {{"app", 2, protoreflect.EnumKind, false}, {"perspective", 3, protoreflect.EnumKind, false}, {"organization_ids", 4, protoreflect.StringKind, true}, {"scopes", 5, protoreflect.MessageKind, true}},
	}
	for requestName, fields := range requests {
		for _, field := range fields {
			assertCounterField(t, messages.ByName(requestName), field.name, field.number, field.kind, field.list)
		}
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
	assertCounterContractDocumentation(t)
}

func assertCounterContractDocumentation(t *testing.T) {
	t.Helper()
	source, err := os.ReadFile("../../../../platform/counter/counter.proto")
	if err != nil {
		t.Fatalf("read counter.proto: %v", err)
	}
	for _, required := range []string{
		"routing/profile context only",
		"owner/BFF MUST bind app to the verified JWT app",
		"reject a caller-scoped",
		"membership/ownership with the source",
		"BFF-selected fields are never authority",
		"CLIENT+BUYER and",
			"PRO+SELLER_ORG",
			"membership_version is 0 iff organization_id is empty",
			"removal/rehire",
			"late event",
	} {
		if !strings.Contains(string(source), required) {
			t.Errorf("counter.proto missing contract documentation %q", required)
		}
	}
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
