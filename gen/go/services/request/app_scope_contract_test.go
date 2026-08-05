package requestv1

import (
	"os"
	"strings"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestPartsRequestReadsCarryAuthoritativeScope(t *testing.T) {
	messages := File_services_request_request_proto.Messages()
	scope := messages.ByName("RequestAccessScope")
	assertRequestScopeField(t, scope, "app", 1, protoreflect.EnumKind)
	assertRequestScopeField(t, scope, "perspective", 2, protoreflect.EnumKind)
	assertRequestScopeField(t, scope, "organization_id", 3, protoreflect.StringKind)
	assertRequestScopeField(t, scope, "membership_version", 4, protoreflect.Int64Kind)

	requestFields := map[protoreflect.Name]protoreflect.FieldNumber{
		"GetRequestRequest":                    2,
		"ListRequestsRequest":                  15,
		"SearchRequestsRequest":                12,
		"GetUserRequestsRequest":               6,
		"GetNewRequestsForOrganizationRequest": 2,
		"MarkRequestAsViewedRequest":           3,
		"IsRequestNewRequest":                  3,
		"DismissRequestRequest":                3,
		"CountUnreadForOrganizationRequest":    9,
		"GetUserRequestCountsRequest":          3,
	}
	for messageName, fieldNumber := range requestFields {
		assertRequestScopeField(t, messages.ByName(messageName), "scope", fieldNumber, protoreflect.MessageKind)
	}
	service := File_services_request_request_proto.Services().ByName("RequestService")
	operations := map[protoreflect.Name]protoreflect.Name{
		"GetRequest":                    "GetRequestRequest",
		"ListRequests":                  "ListRequestsRequest",
		"SearchRequests":                "SearchRequestsRequest",
		"GetUserRequests":               "GetUserRequestsRequest",
		"GetNewRequestsForOrganization": "GetNewRequestsForOrganizationRequest",
		"MarkRequestAsViewed":           "MarkRequestAsViewedRequest",
		"IsRequestNew":                  "IsRequestNewRequest",
		"DismissRequest":                "DismissRequestRequest",
		"CountUnreadForOrganization":    "CountUnreadForOrganizationRequest",
		"GetUserRequestCounts":          "GetUserRequestCountsRequest",
	}
	if len(operations) != len(requestFields) {
		t.Fatal("parts request operation invariant is incomplete")
	}
	for methodName, requestName := range operations {
		method := service.Methods().ByName(methodName)
		if method == nil || method.Input().Name() != requestName || method.Input().Fields().ByName("scope") == nil {
			t.Fatalf("RequestService.%s is not bound to scoped request %s", methodName, requestName)
		}
	}
}

func TestPartsRequestScopeAuthorityIsContractual(t *testing.T) {
	source, err := os.ReadFile("../../../../services/request/request.proto")
	if err != nil {
		t.Fatalf("read request.proto: %v", err)
	}
	for _, required := range []string{
		"JWT app and organization authority wins",
		"CLIENT buyer",
		"PARTNER supplier organization",
		"request user_id and organization_id fields remain filters, never authority",
		"current membership_version",
		"0 is valid only without organization_id",
	} {
		if !strings.Contains(string(source), required) {
			t.Errorf("request.proto missing parts scope rule %q", required)
		}
	}
}

func assertRequestScopeField(t *testing.T, message protoreflect.MessageDescriptor, name protoreflect.Name, number protoreflect.FieldNumber, kind protoreflect.Kind) protoreflect.FieldDescriptor {
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
