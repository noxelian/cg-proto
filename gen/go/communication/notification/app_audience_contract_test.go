package notificationv1

import (
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

	for _, eventName := range []protoreflect.Name{"PushEventPayload", "RealtimeNotificationEventPayload"} {
		event := messages.ByName(eventName)
		for _, field := range []notificationFieldExpectation{
			{"recipient_app", 10, protoreflect.EnumKind},
			{"recipient_perspective", 11, protoreflect.EnumKind},
			{"recipient_organization_id", 12, protoreflect.StringKind},
			{"context_type", 13, protoreflect.StringKind},
			{"context_id", 14, protoreflect.StringKind},
		} {
			assertNotificationField(t, event, field.name, field.number, field.kind)
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
