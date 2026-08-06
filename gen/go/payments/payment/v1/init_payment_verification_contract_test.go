package paymentv1

import (
	"os"
	"strings"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestInitPaymentResponseEchoesOwnerResolvedVerificationFacts(t *testing.T) {
	response := (&InitPaymentResponse{}).ProtoReflect().Descriptor()
	tests := []struct {
		name   protoreflect.Name
		number protoreflect.FieldNumber
		kind   protoreflect.Kind
	}{
		{name: "currency", number: 6, kind: protoreflect.StringKind},
		{name: "organization_id", number: 7, kind: protoreflect.StringKind},
		{name: "entity_type", number: 8, kind: protoreflect.EnumKind},
		{name: "entity_id", number: 9, kind: protoreflect.Int64Kind},
	}
	for _, test := range tests {
		t.Run(string(test.name), func(t *testing.T) {
			field := response.Fields().ByName(test.name)
			if field == nil {
				t.Fatalf("InitPaymentResponse.%s is missing", test.name)
			}
			if field.Number() != test.number || field.Kind() != test.kind {
				t.Fatalf("InitPaymentResponse.%s = field %d (%s), want %d (%s)", test.name, field.Number(), field.Kind(), test.number, test.kind)
			}
		})
	}
	entityType := response.Fields().ByName("entity_type")
	if entityType == nil {
		t.Fatal("InitPaymentResponse.entity_type is missing")
	}
	if entityType.Enum().FullName() != PayEntityType(0).Descriptor().FullName() {
		t.Fatalf("InitPaymentResponse.entity_type enum = %q, want PayEntityType", entityType.Enum().FullName())
	}
}

func TestInitPaymentRequestDoesNotAcceptTransactionType(t *testing.T) {
	request := (&InitPaymentRequest{}).ProtoReflect().Descriptor()
	for _, forbidden := range []protoreflect.Name{"transaction_type", "type"} {
		if field := request.Fields().ByName(forbidden); field != nil {
			t.Fatalf("InitPaymentRequest must not accept caller-supplied %s", field.FullName())
		}
	}
	if got := TransactionType_TRANSACTION_TYPE_CHARGE.Number(); got != 1 {
		t.Fatalf("TRANSACTION_TYPE_CHARGE = %d, want 1", got)
	}
}

func TestInitPaymentVerificationFactsDocumentOwnerAuthority(t *testing.T) {
	source, err := os.ReadFile("../../../../../payments/payment/v1/payment.proto")
	if err != nil {
		t.Fatalf("read payment.proto: %v", err)
	}
	for _, required := range []string{
		"owner-resolved verification facts",
		"never copied from deprecated caller fields",
		"transaction type is always CHARGE",
		"Clients never supply a transaction type",
	} {
		if !strings.Contains(string(source), required) {
			t.Errorf("payment.proto missing InitPayment contract documentation %q", required)
		}
	}
}
