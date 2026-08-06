package paymentv1

import (
	"os"
	"strings"
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestPaymentAudienceEnumValues(t *testing.T) {
	tests := []struct {
		name protoreflect.Name
		want protoreflect.EnumNumber
	}{
		{name: "PAYMENT_AUDIENCE_CLIENT", want: 1},
		{name: "PAYMENT_AUDIENCE_PARTNER", want: 2},
		{name: "PAYMENT_AUDIENCE_SAPP", want: 3},
	}

	for _, test := range tests {
		t.Run(string(test.name), func(t *testing.T) {
			value := PaymentAudience(0).Descriptor().Values().ByName(test.name)
			if value == nil {
				t.Fatalf("PaymentAudience value %q is missing", test.name)
			}
			if got := value.Number(); got != test.want {
				t.Fatalf("PaymentAudience %q = %d, want %d", test.name, got, test.want)
			}
		})
	}
}

func TestProCombinedPayUseCaseValues(t *testing.T) {
	tests := []struct {
		name protoreflect.Name
		want protoreflect.EnumNumber
	}{
		{name: "PAYMENT_USE_CASE_SUBSCRIPTION", want: 3},
		{name: "PAYMENT_USE_CASE_BID_PURCHASE", want: 4},
		{name: "PAYMENT_USE_CASE_CART", want: 5},
	}

	for _, test := range tests {
		t.Run(string(test.name), func(t *testing.T) {
			value := PaymentUseCase(0).Descriptor().Values().ByName(test.name)
			if value == nil {
				t.Fatalf("PaymentUseCase value %q is missing", test.name)
			}
			if got := value.Number(); got != test.want {
				t.Fatalf("PaymentUseCase %q = %d, want %d", test.name, got, test.want)
			}
		})
	}
}

func TestPaymentProviderRouteAudienceFields(t *testing.T) {
	tests := []struct {
		message protoreflect.MessageDescriptor
		number  protoreflect.FieldNumber
	}{
		{message: (&PaymentProviderRoute{}).ProtoReflect().Descriptor(), number: 9},
		{message: (&ListPaymentProviderRoutesRequest{}).ProtoReflect().Descriptor(), number: 1},
		{message: (&SetPaymentProviderRouteRequest{}).ProtoReflect().Descriptor(), number: 4},
	}

	wantEnum := PaymentAudience(0).Descriptor().FullName()
	for _, test := range tests {
		t.Run(string(test.message.Name()), func(t *testing.T) {
			field := test.message.Fields().ByName("audience")
			if field == nil {
				t.Fatal("audience field is missing")
			}
			if got := field.Number(); got != test.number {
				t.Fatalf("audience field number = %d, want %d", got, test.number)
			}
			if got := field.Enum().FullName(); got != wantEnum {
				t.Fatalf("audience enum = %q, want %q", got, wantEnum)
			}
		})
	}
}

func TestInitPaymentRequestHasNoCallerSuppliedAudience(t *testing.T) {
	fields := (&InitPaymentRequest{}).ProtoReflect().Descriptor().Fields()
	if field := fields.ByName("audience"); field != nil {
		t.Fatalf("InitPaymentRequest must derive audience from verified JWT app, found field %q", field.FullName())
	}
}

func TestCreateTransactionCarriesTrustedOwnerAudience(t *testing.T) {
	message := (&CreateTransactionRequest{}).ProtoReflect().Descriptor()
	field := message.Fields().ByName("owner_audience")
	if field == nil {
		t.Fatal("CreateTransactionRequest.owner_audience is missing")
	}
	if got, want := field.Number(), protoreflect.FieldNumber(13); got != want {
		t.Fatalf("owner_audience field number = %d, want %d", got, want)
	}
	if got, want := field.Enum().FullName(), PaymentAudience(0).Descriptor().FullName(); got != want {
		t.Fatalf("owner_audience enum = %q, want %q", got, want)
	}
}

func TestCreateTransactionOwnerAudienceIsNotCallerAuthority(t *testing.T) {
	source, err := os.ReadFile("../../../../../payments/payment/v1/payment.proto")
	if err != nil {
		t.Fatalf("read payment.proto: %v", err)
	}
	for _, required := range []string{
		"accepted only from the exact authenticated owner-service identity",
		"Human and BFF callers must be rejected",
		"empty Auth.App must never default to CLIENT",
	} {
		if !strings.Contains(string(source), required) {
			t.Errorf("payment.proto missing CreateTransaction owner audience contract documentation %q", required)
		}
	}
}

func TestListPaymentProviderRoutesRequiresConcreteAudience(t *testing.T) {
	source, err := os.ReadFile("../../../../../payments/payment/v1/payment.proto")
	if err != nil {
		t.Fatalf("read payment.proto: %v", err)
	}
	for _, required := range []string{
		"audience is REQUIRED and must be a concrete CLIENT, PARTNER, or SAPP value",
		"UNSPECIFIED is invalid and never means list all",
	} {
		if !strings.Contains(string(source), required) {
			t.Errorf("payment.proto missing ListPaymentProviderRoutesRequest contract documentation %q", required)
		}
	}
}
