package userv1

import (
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestAuthorizeOrganizationCapabilityDescriptorContract(t *testing.T) {
	file := File_users_user_user_proto
	service := file.Services().ByName("UserService")
	if service == nil {
		t.Fatal("UserService descriptor not found")
	}

	method := service.Methods().ByName("AuthorizeOrganizationCapability")
	if method == nil {
		t.Fatal("UserService.AuthorizeOrganizationCapability descriptor not found")
	}
	if got, want := method.Input().FullName(), protoreflect.FullName("users.user.v1.AuthorizeOrganizationCapabilityRequest"); got != want {
		t.Fatalf("AuthorizeOrganizationCapability input = %s, want %s", got, want)
	}
	if got, want := method.Output().FullName(), protoreflect.FullName("users.user.v1.AuthorizeOrganizationCapabilityResponse"); got != want {
		t.Fatalf("AuthorizeOrganizationCapability output = %s, want %s", got, want)
	}
	if method.IsStreamingClient() || method.IsStreamingServer() {
		t.Fatalf(
			"AuthorizeOrganizationCapability must be unary, got client_streaming=%t server_streaming=%t",
			method.IsStreamingClient(),
			method.IsStreamingServer(),
		)
	}

	request := file.Messages().ByName("AuthorizeOrganizationCapabilityRequest")
	assertOrganizationCapabilityField(t, request, "organization_id", 1, protoreflect.StringKind, protoreflect.Optional, false)
	assertOrganizationCapabilityField(t, request, "capability", 2, protoreflect.StringKind, protoreflect.Optional, false)
	if got := request.Fields().Len(); got != 2 {
		t.Fatalf("AuthorizeOrganizationCapabilityRequest has %d fields, want exactly 2", got)
	}
	for _, actorField := range []protoreflect.Name{"user_id", "actor_user_id"} {
		if field := request.Fields().ByName(actorField); field != nil {
			t.Fatalf("AuthorizeOrganizationCapabilityRequest must not accept caller-supplied actor field %q", field.FullName())
		}
	}

	response := file.Messages().ByName("AuthorizeOrganizationCapabilityResponse")
	assertOrganizationCapabilityField(t, response, "allowed", 1, protoreflect.BoolKind, protoreflect.Optional, false)
	assertOrganizationCapabilityField(t, response, "actor_user_id", 2, protoreflect.Int64Kind, protoreflect.Optional, false)
	assertOrganizationCapabilityField(t, response, "organization_id", 3, protoreflect.StringKind, protoreflect.Optional, false)
	assertOrganizationCapabilityField(t, response, "capability", 4, protoreflect.StringKind, protoreflect.Optional, false)
	if got := response.Fields().Len(); got != 4 {
		t.Fatalf("AuthorizeOrganizationCapabilityResponse has %d fields, want exactly 4", got)
	}
}

func assertOrganizationCapabilityField(
	t *testing.T,
	message protoreflect.MessageDescriptor,
	name protoreflect.Name,
	number protoreflect.FieldNumber,
	kind protoreflect.Kind,
	cardinality protoreflect.Cardinality,
	hasPresence bool,
) protoreflect.FieldDescriptor {
	t.Helper()
	if message == nil {
		t.Fatalf("message containing %s not found", name)
	}
	field := message.Fields().ByName(name)
	if field == nil {
		t.Fatalf("%s.%s not found", message.Name(), name)
	}
	if got := field.Number(); got != number {
		t.Fatalf("%s.%s number = %d, want %d", message.Name(), name, got, number)
	}
	if got := field.Kind(); got != kind {
		t.Fatalf("%s.%s kind = %s, want %s", message.Name(), name, got, kind)
	}
	if got := field.Cardinality(); got != cardinality {
		t.Fatalf("%s.%s cardinality = %s, want %s", message.Name(), name, got, cardinality)
	}
	if got := field.HasPresence(); got != hasPresence {
		t.Fatalf("%s.%s HasPresence() = %t, want %t", message.Name(), name, got, hasPresence)
	}
	return field
}
