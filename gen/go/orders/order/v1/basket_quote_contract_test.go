package orderv1

import (
	"os"
	"strings"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestBasketQuotePaymentEntityIdentityContract(t *testing.T) {
	requestField := (&GetBasketQuoteRequest{}).ProtoReflect().Descriptor().Fields().ByName("payment_entity_id")
	if requestField == nil {
		t.Fatal("GetBasketQuoteRequest.payment_entity_id is missing")
	}
	if got, want := requestField.Number(), protoreflect.FieldNumber(1); got != want {
		t.Fatalf("GetBasketQuoteRequest.payment_entity_id number = %d, want %d", got, want)
	}
	if got, want := requestField.Kind(), protoreflect.Int64Kind; got != want {
		t.Fatalf("GetBasketQuoteRequest.payment_entity_id kind = %s, want %s", got, want)
	}

	quoteField := (&BasketQuote{}).ProtoReflect().Descriptor().Fields().ByName("payment_entity_id")
	if quoteField == nil {
		t.Fatal("BasketQuote.payment_entity_id is missing")
	}
	if got, want := quoteField.Number(), protoreflect.FieldNumber(1); got != want {
		t.Fatalf("BasketQuote.payment_entity_id number = %d, want %d", got, want)
	}
	if got, want := quoteField.Kind(), protoreflect.Int64Kind; got != want {
		t.Fatalf("BasketQuote.payment_entity_id kind = %s, want %s", got, want)
	}
}

func TestBasketQuotePaymentEntitySemanticsAreContractual(t *testing.T) {
	source, err := os.ReadFile("../../../../../orders/order/v1/order.proto")
	if err != nil {
		t.Fatalf("read order.proto: %v", err)
	}

	for _, required := range []string{
		"GetBasketQuoteRequest.payment_entity_id is the external cart ID",
		"used only to locate and authorize the cart",
		"BasketQuote.payment_entity_id is the owner-minted checkout/payment attempt ID",
		"canonical combined-pay entity_id",
		"may differ from GetBasketQuoteRequest.payment_entity_id",
	} {
		if !strings.Contains(string(source), required) {
			t.Errorf("order.proto missing basket quote identity contract %q", required)
		}
	}
}
