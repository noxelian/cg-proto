package notificationv1

import (
	"os"
	"strings"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestNotificationScopeCoversDeviceAndPreferenceLifecycle(t *testing.T) {
	messages := File_communication_notification_notification_proto.Messages()
	scope := messages.ByName("NotificationScope")
	assertNotificationField(t, scope, "app", 1, protoreflect.EnumKind)
	assertNotificationField(t, scope, "perspective", 2, protoreflect.EnumKind)
	assertNotificationField(t, scope, "organization_id", 3, protoreflect.StringKind)
	assertNotificationField(t, scope, "membership_version", 4, protoreflect.Int64Kind)

	requestFields := map[protoreflect.Name]protoreflect.FieldNumber{
		"RegisterDeviceRequest":    10,
		"UnregisterDeviceRequest":  3,
		"UpdateDeviceRequest":      6,
		"ListUserDevicesRequest":   2,
		"GetPreferencesRequest":    2,
		"UpdatePreferencesRequest": 9,
	}
	for messageName, fieldNumber := range requestFields {
		assertNotificationField(t, messages.ByName(messageName), "scope", fieldNumber, protoreflect.MessageKind)
	}
	service := File_communication_notification_notification_proto.Services().ByName("NotificationService")
	for methodName, requestName := range map[protoreflect.Name]protoreflect.Name{
		"RegisterDevice":    "RegisterDeviceRequest",
		"UnregisterDevice":  "UnregisterDeviceRequest",
		"UpdateDevice":      "UpdateDeviceRequest",
		"ListUserDevices":   "ListUserDevicesRequest",
		"GetPreferences":    "GetPreferencesRequest",
		"UpdatePreferences": "UpdatePreferencesRequest",
	} {
		method := service.Methods().ByName(methodName)
		if method == nil || method.Input().Name() != requestName {
			t.Fatalf("NotificationService.%s request = %v, want %s", methodName, method, requestName)
		}
		if method.Input().Fields().ByName("scope") == nil {
			t.Fatalf("NotificationService.%s request has no scope", methodName)
		}
	}
	assertNotificationField(t, messages.ByName("DeviceInfo"), "scope", 11, protoreflect.MessageKind)
	assertNotificationField(t, messages.ByName("NotificationPreferences"), "scope", 9, protoreflect.MessageKind)
}

func TestNotificationAudienceUsesBoundScopeTuples(t *testing.T) {
	messages := File_communication_notification_notification_proto.Messages()
	pushScopes := assertNotificationField(t, messages.ByName("PushEventPayload"), "recipient_scopes", 16, protoreflect.MessageKind)
	if !pushScopes.IsList() {
		t.Fatal("PushEventPayload.recipient_scopes must be repeated so each app/perspective/organization tuple stays bound")
	}
	assertNotificationField(t, messages.ByName("RealtimeNotificationEventPayload"), "recipient_scope", 17, protoreflect.MessageKind)
}

func TestNotificationScopeAuthorityIsContractual(t *testing.T) {
	source, err := os.ReadFile("../../../../communication/notification/notification.proto")
	if err != nil {
		t.Fatalf("read notification.proto: %v", err)
	}
	for _, required := range []string{
		"single authoritative app + perspective + organization tuple",
		"verified JWT app",
		"Client logout/update cannot mutate Partner registrations",
		"reject conflicting typed and legacy routing",
		"one delivery per recipient_scopes tuple",
		"membership_version is 0 iff organization_id is empty",
		"stale/rehired versions",
	} {
		if !strings.Contains(string(source), required) {
			t.Errorf("notification.proto missing scope authority rule %q", required)
		}
	}
}
