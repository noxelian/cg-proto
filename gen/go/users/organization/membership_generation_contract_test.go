package organizationv1_test

import (
	"os"
	"strings"
	"testing"

	organizationv1 "github.com/4ubak/cg-proto/gen/go/users/organization"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestMembershipGenerationDescriptorContract(t *testing.T) {
	file := organizationv1.File_users_organization_organization_proto
	messages := file.Messages()

	assertMembershipField(t, messages.ByName("Member"), "membership_version", 10, protoreflect.Int64Kind)
	assertMembershipField(t, messages.ByName("OrganizationMembership"), "membership_version", 4, protoreflect.Int64Kind)

	request := messages.ByName("AuthorizeMembershipRequest")
	assertMembershipField(t, request, "user_id", 1, protoreflect.Int64Kind)
	assertMembershipField(t, request, "organization_id", 2, protoreflect.StringKind)
	assertMembershipField(t, request, "membership_version", 3, protoreflect.Int64Kind)

	response := messages.ByName("AuthorizeMembershipResponse")
	assertMembershipField(t, response, "authorized", 1, protoreflect.BoolKind)
	assertMembershipField(t, response, "current_membership_version", 2, protoreflect.Int64Kind)
	assertMembershipField(t, response, "member", 3, protoreflect.MessageKind)

	listRequest := messages.ByName("ListEligibleMembersRequest")
	assertMembershipField(t, listRequest, "organization_id", 1, protoreflect.StringKind)
	assertMembershipField(t, listRequest, "page_size", 2, protoreflect.Int32Kind)
	assertMembershipField(t, listRequest, "page_token", 3, protoreflect.StringKind)

	recipient := messages.ByName("EligibleMemberRecipient")
	assertMembershipField(t, recipient, "user_id", 1, protoreflect.Int64Kind)
	assertMembershipField(t, recipient, "membership_version", 2, protoreflect.Int64Kind)

	listResponse := messages.ByName("ListEligibleMembersResponse")
	recipients := assertMembershipField(t, listResponse, "recipients", 1, protoreflect.MessageKind)
	if !recipients.IsList() {
		t.Fatal("ListEligibleMembersResponse.recipients must be repeated")
	}
	assertMembershipField(t, listResponse, "next_page_token", 2, protoreflect.StringKind)

	event := messages.ByName("OrganizationMembershipChangedEvent")
	for _, expected := range []membershipFieldExpectation{
		{"event_id", 1, protoreflect.StringKind},
		{"organization_id", 2, protoreflect.StringKind},
		{"user_id", 3, protoreflect.Int64Kind},
		{"membership_version", 4, protoreflect.Int64Kind},
		{"old_status", 5, protoreflect.EnumKind},
		{"new_status", 6, protoreflect.EnumKind},
		{"event_type", 7, protoreflect.EnumKind},
		{"occurred_at", 8, protoreflect.MessageKind},
	} {
		assertMembershipField(t, event, expected.name, expected.number, expected.kind)
	}

	service := file.Services().ByName("OrganizationService")
	if service == nil {
		t.Fatal("OrganizationService descriptor not found")
	}
	for _, methodName := range []protoreflect.Name{"AuthorizeMembership", "ListEligibleMembers"} {
		if service.Methods().ByName(methodName) == nil {
			t.Fatalf("OrganizationService.%s not found", methodName)
		}
	}
}

func TestMembershipGenerationLifecycleEnumContract(t *testing.T) {
	enum := organizationv1.File_users_organization_organization_proto.Enums().ByName("OrganizationMembershipChangeType")
	if enum == nil {
		t.Fatal("OrganizationMembershipChangeType descriptor not found")
	}
	want := map[protoreflect.Name]protoreflect.EnumNumber{
		"ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_UNSPECIFIED":  0,
		"ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_ADD":          1,
		"ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_FIRE":         2,
		"ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_REHIRE":       3,
		"ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_ROLE_CHANGED": 4,
	}
	for name, number := range want {
		value := enum.Values().ByName(name)
		if value == nil || value.Number() != number {
			t.Fatalf("%s = %v, want %d", name, value, number)
		}
	}
}

func TestMembershipGenerationSourceDocumentsSecurityAndLifecycle(t *testing.T) {
	source, err := os.ReadFile("../../../../users/organization/organization.proto")
	if err != nil {
		t.Fatalf("read organization.proto: %v", err)
	}
	for _, required := range []string{
		"first active membership is 1",
		"increments only when a fired member",
		"stale generation fails",
		"JWT organization claims are context, never",
		"MUST NOT be exposed through public BFF",
		"no implicit",
		"recipient cap",
		"revalidate a pending recipient",
		"negative value is",
		"integrity-protected",
		"cryptographically bound to organization_id",
		"stable (user_id, member_id)",
		"cross-organization token is INVALID_ARGUMENT",
		"transactional-outbox/Kafka contract",
		"compat/users nests this payload",
		"authoritative prior membership state",
		"never use protojson",
	} {
		if !strings.Contains(string(source), required) {
			t.Errorf("organization.proto missing contract documentation %q", required)
		}
	}
}

type membershipFieldExpectation struct {
	name   protoreflect.Name
	number protoreflect.FieldNumber
	kind   protoreflect.Kind
}

func assertMembershipField(t *testing.T, message protoreflect.MessageDescriptor, name protoreflect.Name, number protoreflect.FieldNumber, kind protoreflect.Kind) protoreflect.FieldDescriptor {
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
