package notificationv1

import (
	"os"
	"strings"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestAppAudienceContract(t *testing.T) {
	enums := File_communication_notification_notification_proto.Enums()
	assertNotificationEnumValue(t, enums.ByName("NotificationApp"), "NOTIFICATION_APP_CLIENT", 1)
	assertNotificationEnumValue(t, enums.ByName("NotificationApp"), "NOTIFICATION_APP_PRO", 2)
	assertNotificationEnumValue(t, enums.ByName("NotificationPerspective"), "NOTIFICATION_PERSPECTIVE_BUYER", 1)
	assertNotificationEnumValue(t, enums.ByName("NotificationPerspective"), "NOTIFICATION_PERSPECTIVE_SELLER_ORG", 2)
	assertNotificationEnumValue(t, enums.ByName("NotificationPerspective"), "NOTIFICATION_PERSPECTIVE_SELLER_USER", 3)
	assertNotificationEnumValue(t, enums.ByName("NotificationPerspective"), "NOTIFICATION_PERSPECTIVE_SUPPORT", 4)

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

func assertNotificationField(t *testing.T, message protoreflect.MessageDescriptor, name protoreflect.Name, number protoreflect.FieldNumber, kind protoreflect.Kind) {
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
