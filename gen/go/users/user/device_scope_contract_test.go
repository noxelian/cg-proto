package userv1

import (
	"os"
	"strings"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestLegacyUserDeviceAndSettingsRPCsAreAppScopedAcrossLifecycle(t *testing.T) {
	messages := File_users_user_user_proto.Messages()
	scope := messages.ByName("UserAppScope")
	assertUserDeviceField(t, scope, "app", 1, protoreflect.EnumKind)
	assertUserDeviceField(t, scope, "perspective", 2, protoreflect.EnumKind)
	assertUserDeviceField(t, scope, "organization_id", 3, protoreflect.StringKind)
	assertUserDeviceField(t, scope, "membership_version", 4, protoreflect.Int64Kind)
	perspective := File_users_user_user_proto.Enums().ByName("UserPerspective").Values().ByName("USER_PERSPECTIVE_BUYER_ORG")
	if perspective == nil || perspective.Number() != 4 {
		t.Fatalf("USER_PERSPECTIVE_BUYER_ORG = %v, want 4", perspective)
	}

	for messageName, fieldNumber := range map[protoreflect.Name]protoreflect.FieldNumber{
		"GetSettingsRequest":      1,
		"UpdateSettingsRequest":   7,
		"RegisterDeviceRequest":   4,
		"UnregisterDeviceRequest": 2,
		"GetDevicesRequest":       1,
	} {
		assertUserDeviceField(t, messages.ByName(messageName), "scope", fieldNumber, protoreflect.MessageKind)
	}
	assertUserDeviceField(t, messages.ByName("UserSettings"), "scope", 7, protoreflect.MessageKind)
	assertUserDeviceField(t, messages.ByName("Device"), "scope", 7, protoreflect.MessageKind)
}

func TestLegacyUserDeviceScopeAuthorityIsContractual(t *testing.T) {
	source, err := os.ReadFile("../../../../users/user/user.proto")
	if err != nil {
		t.Fatalf("read user.proto: %v", err)
	}
	for _, required := range []string{
		"verified JWT app",
		"Client logout cannot unregister a Partner device",
		"scope takes precedence",
		"reject a conflict",
		"membership_version is 0 iff organization_id",
		"removal/rehire",
	} {
		if !strings.Contains(string(source), required) {
			t.Errorf("user.proto missing device scope authority rule %q", required)
		}
	}
}

func assertUserDeviceField(t *testing.T, message protoreflect.MessageDescriptor, name protoreflect.Name, number protoreflect.FieldNumber, kind protoreflect.Kind) protoreflect.FieldDescriptor {
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
