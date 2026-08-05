package bidv1

import (
	"os"
	"strings"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestPartsBidReadsCarryAuthoritativeScope(t *testing.T) {
	messages := File_services_bid_bid_proto.Messages()
	scope := messages.ByName("BidAccessScope")
	assertBidScopeField(t, scope, "app", 1, protoreflect.EnumKind)
	assertBidScopeField(t, scope, "perspective", 2, protoreflect.EnumKind)
	assertBidScopeField(t, scope, "organization_id", 3, protoreflect.StringKind)
	assertBidScopeField(t, scope, "membership_version", 4, protoreflect.Int64Kind)

	requestFields := map[protoreflect.Name]protoreflect.FieldNumber{
		"GetBidRequest":                        2,
		"GetBidForBuyerRequest":                3,
		"ListBidsForBuyerRequest":              3,
		"HasAcceptedBidForOrganizationRequest": 3,
		"ListBidsRequest":                      7,
		"GetBidsByRequestRequest":              5,
		"GetBidsByOrganizationRequest":         5,
		"GetBidPartPricesRequest":              2,
		"MarkBidReadRequest":                   4,
		"GetRequestResponsesSummaryRequest":    3,
	}
	for messageName, fieldNumber := range requestFields {
		assertBidScopeField(t, messages.ByName(messageName), "scope", fieldNumber, protoreflect.MessageKind)
	}
	service := File_services_bid_bid_proto.Services().ByName("BidService")
	operations := map[protoreflect.Name]protoreflect.Name{
		"GetBid":                        "GetBidRequest",
		"GetBidForBuyer":                "GetBidForBuyerRequest",
		"ListBidsForBuyer":              "ListBidsForBuyerRequest",
		"HasAcceptedBidForOrganization": "HasAcceptedBidForOrganizationRequest",
		"ListBids":                      "ListBidsRequest",
		"GetBidsByRequest":              "GetBidsByRequestRequest",
		"GetBidsByOrganization":         "GetBidsByOrganizationRequest",
		"GetBidPartPrices":              "GetBidPartPricesRequest",
		"MarkBidRead":                   "MarkBidReadRequest",
		"GetRequestResponsesSummary":    "GetRequestResponsesSummaryRequest",
	}
	if len(operations) != len(requestFields) {
		t.Fatal("parts bid operation invariant is incomplete")
	}
	for methodName, requestName := range operations {
		method := service.Methods().ByName(methodName)
		if method == nil || method.Input().Name() != requestName || method.Input().Fields().ByName("scope") == nil {
			t.Fatalf("BidService.%s is not bound to scoped request %s", methodName, requestName)
		}
	}
}

func TestPartsBidScopeAuthorityIsContractual(t *testing.T) {
	source, err := os.ReadFile("../../../../services/bid/bid.proto")
	if err != nil {
		t.Fatalf("read bid.proto: %v", err)
	}
	for _, required := range []string{
		"JWT app and organization authority wins",
		"CLIENT buyer",
		"PARTNER supplier organization",
		"user_id and organization_id remain compatibility filters, never authority",
		"current membership_version",
		"0 only when organization_id is empty",
	} {
		if !strings.Contains(string(source), required) {
			t.Errorf("bid.proto missing parts scope rule %q", required)
		}
	}
}

func assertBidScopeField(t *testing.T, message protoreflect.MessageDescriptor, name protoreflect.Name, number protoreflect.FieldNumber, kind protoreflect.Kind) protoreflect.FieldDescriptor {
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
