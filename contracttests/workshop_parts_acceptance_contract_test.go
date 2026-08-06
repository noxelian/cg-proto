package contracttests

import (
	"reflect"
	"testing"

	workshopv1 "github.com/4ubak/cg-proto/gen/go/workshop"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestPartsAcceptanceEnumNumbersAreAdditiveAndStable(t *testing.T) {
	want := map[protoreflect.Name]protoreflect.EnumNumber{
		"PARTS_ACCEPTANCE_STATE_UNSPECIFIED":                   0,
		"PARTS_ACCEPTANCE_STATE_NOT_REQUIRED":                  1,
		"PARTS_ACCEPTANCE_STATE_WAITING_DELIVERY":              2,
		"PARTS_ACCEPTANCE_STATE_DELIVERED_AWAITING_ACCEPTANCE": 3,
		"PARTS_ACCEPTANCE_STATE_ACCEPTED":                      4,
	}
	enum := workshopv1.PartsAcceptanceState(0).Descriptor()
	if enum.Values().Len() != len(want) {
		t.Fatalf("PartsAcceptanceState values = %d, want %d", enum.Values().Len(), len(want))
	}
	for name, number := range want {
		value := enum.Values().ByName(name)
		if value == nil || value.Number() != number {
			t.Fatalf("PartsAcceptanceState.%s = %v, want %d", name, value, number)
		}
	}
}

func TestPartsAcceptanceProjectionDescriptorIsExact(t *testing.T) {
	repairOrder := (&workshopv1.RepairOrder{}).ProtoReflect().Descriptor()
	projectionField := assertPartsField(t, repairOrder, "parts_acceptance", 61, protoreflect.MessageKind, protoreflect.Optional, true)
	if got := projectionField.Message().FullName(); got != "workshop.v1.PartsAcceptance" {
		t.Fatalf("RepairOrder.parts_acceptance type = %s, want workshop.v1.PartsAcceptance", got)
	}

	projection := (&workshopv1.PartsAcceptance{}).ProtoReflect().Descriptor()
	assertExactPartsFields(t, projection, []partsFieldExpectation{
		{name: "state", number: 1, kind: protoreflect.EnumKind, cardinality: protoreflect.Optional, presence: false},
		{name: "delivered_at", number: 2, kind: protoreflect.MessageKind, cardinality: protoreflect.Optional, presence: true},
		{name: "accepted_at", number: 3, kind: protoreflect.MessageKind, cardinality: protoreflect.Optional, presence: true},
		{name: "accepted_by_user_id", number: 4, kind: protoreflect.Int64Kind, cardinality: protoreflect.Optional, presence: false},
	})

	kanban := (&workshopv1.KanbanColumn{}).ProtoReflect().Descriptor()
	orders := assertPartsField(t, kanban, "orders", 4, protoreflect.MessageKind, protoreflect.Repeated, false)
	if got := orders.Message().FullName(); got != "workshop.v1.RepairOrder" {
		t.Fatalf("KanbanColumn.orders type = %s, want workshop.v1.RepairOrder", got)
	}
}

func TestAcceptDeliveredPartsCommandDescriptorIsNarrowUnaryAndPresenceSafe(t *testing.T) {
	request := (&workshopv1.AcceptDeliveredPartsRequest{}).ProtoReflect().Descriptor()
	assertExactPartsFields(t, request, []partsFieldExpectation{
		{name: "repair_order_id", number: 1, kind: protoreflect.Int64Kind, cardinality: protoreflect.Optional, presence: false},
		{name: "idempotency_key", number: 2, kind: protoreflect.StringKind, cardinality: protoreflect.Optional, presence: false},
		{name: "note", number: 3, kind: protoreflect.StringKind, cardinality: protoreflect.Optional, presence: true},
	})
	if note := request.Fields().ByName("note"); note == nil || !note.HasOptionalKeyword() {
		t.Fatal("AcceptDeliveredPartsRequest.note must remain proto3 optional")
	}

	absent := &workshopv1.AcceptDeliveredPartsRequest{RepairOrderId: 42, IdempotencyKey: "parts-42"}
	empty := &workshopv1.AcceptDeliveredPartsRequest{
		RepairOrderId: 42, IdempotencyKey: "parts-42", Note: proto.String(""),
	}
	if absent.Note != nil || empty.Note == nil || proto.Equal(absent, empty) {
		t.Fatal("absent and present-empty parts acceptance notes must remain distinct")
	}

	response := (&workshopv1.AcceptDeliveredPartsResponse{}).ProtoReflect().Descriptor()
	assertExactPartsFields(t, response, []partsFieldExpectation{
		{name: "order", number: 1, kind: protoreflect.MessageKind, cardinality: protoreflect.Optional, presence: true},
		{name: "already_accepted", number: 2, kind: protoreflect.BoolKind, cardinality: protoreflect.Optional, presence: false},
	})

	service := workshopv1.File_workshop_workshop_proto.Services().ByName("WorkshopService")
	method := service.Methods().ByName("AcceptDeliveredParts")
	if method == nil {
		t.Fatal("WorkshopService.AcceptDeliveredParts is missing")
	}
	if method.IsStreamingClient() || method.IsStreamingServer() {
		t.Fatal("WorkshopService.AcceptDeliveredParts must remain unary")
	}
	if got := method.Input().FullName(); got != "workshop.v1.AcceptDeliveredPartsRequest" {
		t.Fatalf("AcceptDeliveredParts input = %s", got)
	}
	if got := method.Output().FullName(); got != "workshop.v1.AcceptDeliveredPartsResponse" {
		t.Fatalf("AcceptDeliveredParts output = %s", got)
	}
}

func TestPartsAcceptanceAuthorityAndReplaySemanticsArePinned(t *testing.T) {
	workshopSource := readContractFile(t, "workshop/workshop.proto")
	adr := readContractFile(t, "docs/adr/0001-workshop-v2-contracts.md")

	command := contractBlock(t, workshopSource, "message AcceptDeliveredPartsRequest", "message AcceptDeliveredPartsResponse")
	assertContainsAll(t, command,
		"Only DELIVERED_AWAITING_ACCEPTANCE may transition to ACCEPTED",
		"event may establish delivered_at",
		"it is never acceptance",
		"workshop:accept_delivered_parts",
		"Machine/service principals",
		"read-only AI workshop master",
		"exact same-fingerprint replay",
		"already_accepted=true",
		"ALREADY_EXISTS without mutation",
		"atomically binds that key as an alias",
	)
	assertContainsAll(t, adr,
		"`AcceptDeliveredParts` is the only transition",
		"derives `accepted_by_user_id` from the authenticated principal",
		"body contains none of those authority fields",
		"absent `parts_acceptance` projection or `UNSPECIFIED` state preserves legacy behavior",
		"Read-only AI consumers may observe this projection but cannot invoke the acceptance command",
		"`issuance_closing`",
	)
}

type partsFieldExpectation struct {
	name        protoreflect.Name
	number      protoreflect.FieldNumber
	kind        protoreflect.Kind
	cardinality protoreflect.Cardinality
	presence    bool
}

func assertExactPartsFields(t *testing.T, message protoreflect.MessageDescriptor, want []partsFieldExpectation) {
	t.Helper()
	fields := message.Fields()
	if fields.Len() != len(want) {
		t.Fatalf("%s fields = %d, want %d", message.FullName(), fields.Len(), len(want))
	}
	gotNames := make([]string, 0, fields.Len())
	wantNames := make([]string, 0, len(want))
	for i, expected := range want {
		gotNames = append(gotNames, string(fields.Get(i).Name()))
		wantNames = append(wantNames, string(expected.name))
		assertPartsField(t, message, expected.name, expected.number, expected.kind, expected.cardinality, expected.presence)
	}
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("%s fields = %v, want %v", message.FullName(), gotNames, wantNames)
	}
}

func assertPartsField(
	t *testing.T,
	message protoreflect.MessageDescriptor,
	name protoreflect.Name,
	number protoreflect.FieldNumber,
	kind protoreflect.Kind,
	cardinality protoreflect.Cardinality,
	presence bool,
) protoreflect.FieldDescriptor {
	t.Helper()
	field := message.Fields().ByName(name)
	if field == nil {
		t.Fatalf("%s.%s is missing", message.FullName(), name)
	}
	if field.Number() != number || field.Kind() != kind || field.Cardinality() != cardinality || field.HasPresence() != presence {
		t.Fatalf("%s.%s = number %d kind %s cardinality %s presence %t; want %d %s %s %t",
			message.FullName(), name, field.Number(), field.Kind(), field.Cardinality(), field.HasPresence(),
			number, kind, cardinality, presence)
	}
	return field
}
