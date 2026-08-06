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
	assertRequestEnumValue(t, File_services_request_request_proto.Enums().ByName("RequestAccessPerspective"), "REQUEST_ACCESS_PERSPECTIVE_BUYER_ORG", 4)

	requestFields := map[protoreflect.Name]protoreflect.FieldNumber{
		"CreateRequestRequest":                 28,
		"GetRequestRequest":                    2,
		"UpdateRequestRequest":                 16,
		"DeleteRequestRequest":                 3,
		"ListRequestsRequest":                  15,
		"SearchRequestsRequest":                12,
		"ChangeStatusRequest":                  4,
		"GetUserRequestsRequest":               6,
		"GetNewRequestsForOrganizationRequest": 2,
		"MarkRequestAsViewedRequest":           3,
		"IsRequestNewRequest":                  3,
		"DismissRequestRequest":                3,
		"CountUnreadForOrganizationRequest":    9,
		"GetUserRequestCountsRequest":          3,
		"PauseRequestRequest":                  4,
		"UnpauseRequestRequest":                3,
		"SelectRequestBidRequest":              5,
		"CloseRequestRequest":                  4,
		"ReserveRequestBidSelectionRequest":    5,
		"CancelRequestBidSelectionRequest":     5,
	}
	for messageName, fieldNumber := range requestFields {
		assertRequestScopeField(t, messages.ByName(messageName), "scope", fieldNumber, protoreflect.MessageKind)
	}
	service := File_services_request_request_proto.Services().ByName("RequestService")
	operations := map[protoreflect.Name]protoreflect.Name{
		"CreateRequest":                 "CreateRequestRequest",
		"GetRequest":                    "GetRequestRequest",
		"UpdateRequest":                 "UpdateRequestRequest",
		"DeleteRequest":                 "DeleteRequestRequest",
		"ListRequests":                  "ListRequestsRequest",
		"SearchRequests":                "SearchRequestsRequest",
		"ChangeStatus":                  "ChangeStatusRequest",
		"GetUserRequests":               "GetUserRequestsRequest",
		"GetNewRequestsForOrganization": "GetNewRequestsForOrganizationRequest",
		"MarkRequestAsViewed":           "MarkRequestAsViewedRequest",
		"IsRequestNew":                  "IsRequestNewRequest",
		"DismissRequest":                "DismissRequestRequest",
		"CountUnreadForOrganization":    "CountUnreadForOrganizationRequest",
		"GetUserRequestCounts":          "GetUserRequestCountsRequest",
		"PauseRequest":                  "PauseRequestRequest",
		"UnpauseRequest":                "UnpauseRequestRequest",
		"SelectRequestBid":              "SelectRequestBidRequest",
		"CloseRequest":                  "CloseRequestRequest",
		"ReserveRequestBidSelection":    "ReserveRequestBidSelectionRequest",
		"CancelRequestBidSelection":     "CancelRequestBidSelectionRequest",
	}
	serviceOnly := map[protoreflect.Name]protoreflect.Name{
		"ListPublishedPreviews":         "ListPublishedPreviewsRequest",
		"IncrementViews":                "IncrementViewsRequest",
		"GetSuggestions":                "GetSuggestionsRequest",
		"ClassifyRequest":               "ClassifyRequestRequest",
		"GetRequestForClassification":   "GetRequestForClassificationRequest",
		"GetInsurancePayoutTerms":       "GetInsurancePayoutTermsRequest",
		"GetRequestEligibilityInfo":     "GetRequestEligibilityInfoRequest",
		"GetRequestInsuranceInfo":       "GetRequestInsuranceInfoRequest",
		"ClaimInsuranceRequest":         "ClaimInsuranceRequestRequest",
		"CompleteInsuranceRequest":      "CompleteInsuranceRequestRequest",
		"IsOrgTargeted":                 "IsOrgTargetedRequest",
		"PrepareRequestEscrow":          "PrepareRequestEscrowRequest",
		"AuthorizeRequestEscrowCapture": "AuthorizeRequestEscrowCaptureRequest",
		"MarkRequestEscrowState":        "MarkRequestEscrowStateRequest",
		"ListRequestUserIDs":            "ListRequestUserIDsRequest",
	}
	if service.Methods().Len() != len(operations)+len(serviceOnly) {
		t.Fatalf("RequestService has %d methods; scoped=%d separate-boundary=%d", service.Methods().Len(), len(operations), len(serviceOnly))
	}
	for methodName, requestName := range serviceOnly {
		method := service.Methods().ByName(methodName)
		if method == nil || method.Input().Name() != requestName || method.Input().Fields().ByName("scope") != nil {
			t.Fatalf("RequestService.%s separate authorization boundary changed", methodName)
		}
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

func assertRequestEnumValue(t *testing.T, enum protoreflect.EnumDescriptor, name protoreflect.Name, number protoreflect.EnumNumber) {
	t.Helper()
	if enum == nil || enum.Values().ByName(name) == nil || enum.Values().ByName(name).Number() != number {
		t.Fatalf("enum value %s=%d not found", name, number)
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
