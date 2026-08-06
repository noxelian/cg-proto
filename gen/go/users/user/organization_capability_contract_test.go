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

	request := file.Messages().ByName("AuthorizeOrganizationCapabilityRequest")
	assertOrganizationCapabilityField(t, request, "organization_id", 1, protoreflect.StringKind)
	assertOrganizationCapabilityField(t, request, "capability", 2, protoreflect.StringKind)
	if got := request.Fields().Len(); got != 2 {
		t.Fatalf("AuthorizeOrganizationCapabilityRequest has %d fields, want exactly 2", got)
	}
	for _, actorField := range []protoreflect.Name{"user_id", "actor_user_id"} {
		if field := request.Fields().ByName(actorField); field != nil {
			t.Fatalf("AuthorizeOrganizationCapabilityRequest must not accept caller-supplied actor field %q", field.FullName())
		}
	}

	response := file.Messages().ByName("AuthorizeOrganizationCapabilityResponse")
	assertOrganizationCapabilityField(t, response, "allowed", 1, protoreflect.BoolKind)
	assertOrganizationCapabilityField(t, response, "actor_user_id", 2, protoreflect.Int64Kind)
	assertOrganizationCapabilityField(t, response, "organization_id", 3, protoreflect.StringKind)
	assertOrganizationCapabilityField(t, response, "capability", 4, protoreflect.StringKind)
	if got := response.Fields().Len(); got != 4 {
		t.Fatalf("AuthorizeOrganizationCapabilityResponse has %d fields, want exactly 4", got)
	}
}

func assertOrganizationCapabilityField(t *testing.T, message protoreflect.MessageDescriptor, name protoreflect.Name, number protoreflect.FieldNumber, kind protoreflect.Kind) protoreflect.FieldDescriptor {
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
