package contracttests

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	crmv1 "github.com/4ubak/cg-proto/gen/go/crm"
	_ "github.com/4ubak/cg-proto/gen/go/users/garage"
	workshopv1 "github.com/4ubak/cg-proto/gen/go/workshop"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func TestWorkshopIntakeProjectionContractsAreNarrowAndBound(t *testing.T) {
	assertMethod(t, "crm.v1.CRMService", "GetWorkshopIntakeProjection")
	assertFields(t, "crm.v1.GetWorkshopIntakeProjectionRequest", []string{
		"organization_id",
		"workshop_id",
		"source_event_id",
		"crm_deal_id",
	})
	assertFields(t, "crm.v1.WorkshopIntakeProjection", []string{
		"organization_id",
		"workshop_id",
		"source_event_id",
		"crm_deal_id",
		"crm_pipeline_id",
		"crm_source_stage_system_code",
		"user_id",
		"garage_car_id",
		"client_name",
		"client_phone",
		"license_plate",
		"vin",
		"car_color",
		"car_year",
		"mark_id",
		"model_id",
		"description",
		"crm_deal_title",
	})

	assertMethod(t, "users.garage.v1.GarageService", "GetWorkshopIntakeParty")
	assertFields(t, "users.garage.v1.GetWorkshopIntakePartyRequest", []string{
		"organization_id",
		"workshop_id",
		"source_event_id",
		"crm_deal_id",
		"user_id",
		"garage_car_id",
	})
	assertFields(t, "users.garage.v1.WorkshopIntakePartyProjection", []string{
		"organization_id",
		"workshop_id",
		"source_event_id",
		"crm_deal_id",
		"user_id",
		"garage_car_id",
		"client_name",
		"client_phone",
		"license_plate",
		"vin",
		"car_color",
		"car_year",
		"mark_id",
		"model_id",
	})
}

func TestCreateIntakeOrderFromCRMRequestRemainsIDsOnly(t *testing.T) {
	fields := (&workshopv1.CreateIntakeOrderFromCRMRequest{}).ProtoReflect().Descriptor().Fields()
	got := make([]string, 0, fields.Len())
	for i := 0; i < fields.Len(); i++ {
		got = append(got, string(fields.Get(i).Name()))
	}
	want := []string{"idempotency_key", "source_event_id", "workshop_id", "crm_deal_id"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CreateIntakeOrderFromCRMRequest fields = %v, want %v", got, want)
	}
}

func TestWorkshopV2ReplayAndAuthorizationSemanticsArePinned(t *testing.T) {
	workshopSource := readContractFile(t, "workshop/workshop.proto")
	crmSource := readContractFile(t, "crm/crm.proto")
	garageSource := readContractFile(t, "users/garage/garage.proto")
	adr := readContractFile(t, "docs/adr/0001-workshop-v2-contracts.md")

	assertContainsAll(t, workshopSource,
		"(authoritative organization_id, workshop_id,",
		"WorkshopService.CreateIntakeOrderFromCRM)",
		"(source_event_id, workshop_id, crm_deal_id)",
		"(authoritative organization_id, authoritative workshop_id,",
		"WorkshopService.AcceptVehicle)",
		"(repair_order_id, note presence, note UTF-8 bytes)",
	)
	assertContainsAll(t, crmSource,
		"UserID=0, phone=device_id=\"cg-workshop\"",
		"organization_id, workshop_id, source_event_id and crm_deal_id",
		"durable CRM workshop-handoff/outbox record",
	)
	assertContainsAll(t, garageSource,
		"UserID=0, phone=device_id=\"cg-crm\"",
		"car.user_id == user_id",
		"returns only the user/car fields required for workshop intake",
	)
	assertContainsAll(t, adr,
		"authenticate and bind the tenant/service scope before any deduplication lookup",
		"look up the durable command record by the immutable command identities before reading CRM or cg-users",
		"matching fingerprint returns the stored result without re-reading CRM or cg-users",
		"conflicting fingerprint returns gRPC `ALREADY_EXISTS`",
		"Only the first execution may resolve the current authoritative CRM/user/car sources",
		"atomically persists the repair order, deduplication records, intake audit and outbox event",
		"An outage or deletion of a CRM, user or car source after first success cannot break an exact replay",
		"no second intake audit or outbox event",
		"For both commands, replaying the same semantics returns the same logical repair order",
		"without inserting a second audit row or publishing a second outbox event",
	)
}

func TestWorkshopV2ExactReplayBindsEveryNewScopedKeyAtomically(t *testing.T) {
	workshopSource := readContractFile(t, "workshop/workshop.proto")
	adr := readContractFile(t, "docs/adr/0001-workshop-v2-contracts.md")

	acceptVehicle := contractBlock(t, workshopSource, "message AcceptVehicleRequest", "message AcceptVehicleResponse")
	createIntake := contractBlock(t, workshopSource, "message CreateIntakeOrderFromCRMRequest", "message CreateIntakeOrderFromCRMResponse")
	for name, command := range map[string]string{
		"AcceptVehicle":            acceptVehicle,
		"CreateIntakeOrderFromCRM": createIntake,
	} {
		t.Run(name, func(t *testing.T) {
			assertContainsAll(t, command,
				"previously unseen scoped idempotency key",
				"MUST, before returning, atomically persist that key as an alias",
				"stored immutable fingerprint and result",
				"already bound to a different fingerprint or result",
				"ALREADY_EXISTS with no mutation",
				"same uniqueness and transactional serialization",
			)
		})
	}

	assertContainsAll(t, adr,
		"D1/K1 -> D1/K2 -> D2/K2",
		"D1/K2 must bind K2 to R1 before returning R1",
		"D2/K2 must then return `ALREADY_EXISTS` without mutation",
		"concurrent D1/K2 and D2/K2 cannot both succeed",
		"must not create a second order, intake or arrival audit, outbox event, or source-system read",
	)
}

func TestAcceptVehicleNotePresenceAndAdditiveNumbers(t *testing.T) {
	absent := &workshopv1.AcceptVehicleRequest{RepairOrderId: 42, IdempotencyKey: "arrival-42"}
	empty := &workshopv1.AcceptVehicleRequest{
		RepairOrderId:  42,
		IdempotencyKey: "arrival-42",
		Note:           proto.String(""),
	}
	if absent.Note != nil || empty.Note == nil || proto.Equal(absent, empty) {
		t.Fatal("absent and present-empty note must remain distinct fingerprint inputs")
	}

	if got := int32(workshopv1.RepairStatus_REPAIR_STATUS_CLOSED_BY_CLIENT); got != 13 {
		t.Fatalf("legacy CLOSED_BY_CLIENT number = %d, want 13", got)
	}
	if got := int32(workshopv1.RepairStatus_REPAIR_STATUS_INSPECTION); got != 14 {
		t.Fatalf("INSPECTION number = %d, want 14", got)
	}
	if got := int32(workshopv1.PaymentMethod_PAYMENT_METHOD_BANK_TRANSFER); got != 5 {
		t.Fatalf("BANK_TRANSFER number = %d, want 5", got)
	}
	if got := (&crmv1.Stage{}).ProtoReflect().Descriptor().Fields().ByName("system_code").Number(); got != 16 {
		t.Fatalf("Stage.system_code number = %d, want 16", got)
	}
}

func assertMethod(t *testing.T, serviceName protoreflect.FullName, methodName protoreflect.Name) {
	t.Helper()
	desc, err := protoregistry.GlobalFiles.FindDescriptorByName(serviceName)
	if err != nil {
		t.Fatalf("find service %s: %v", serviceName, err)
	}
	service, ok := desc.(protoreflect.ServiceDescriptor)
	if !ok {
		t.Fatalf("%s is %T, want service descriptor", serviceName, desc)
	}
	if service.Methods().ByName(methodName) == nil {
		t.Fatalf("service %s missing method %s", serviceName, methodName)
	}
}

func assertFields(t *testing.T, messageName protoreflect.FullName, want []string) {
	t.Helper()
	desc, err := protoregistry.GlobalFiles.FindDescriptorByName(messageName)
	if err != nil {
		t.Fatalf("find message %s: %v", messageName, err)
	}
	message, ok := desc.(protoreflect.MessageDescriptor)
	if !ok {
		t.Fatalf("%s is %T, want message descriptor", messageName, desc)
	}
	fields := message.Fields()
	got := make([]string, 0, fields.Len())
	for i := 0; i < fields.Len(); i++ {
		got = append(got, string(fields.Get(i).Name()))
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s fields = %v, want %v", messageName, got, want)
	}
}

func readContractFile(t *testing.T, relative string) string {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve contract test source path")
	}
	root := filepath.Dir(filepath.Dir(current))
	payload, err := os.ReadFile(filepath.Join(root, relative))
	if err != nil {
		t.Fatalf("read %s: %v", relative, err)
	}
	return string(payload)
}

func assertContainsAll(t *testing.T, source string, fragments ...string) {
	t.Helper()
	normalizedSource := normalizeWhitespace(source)
	for _, fragment := range fragments {
		if !strings.Contains(normalizedSource, normalizeWhitespace(fragment)) {
			t.Errorf("contract text missing %q", fragment)
		}
	}
}

func contractBlock(t *testing.T, source, start, end string) string {
	t.Helper()
	startIndex := strings.Index(source, start)
	if startIndex < 0 {
		t.Fatalf("contract block start %q not found", start)
	}
	endIndex := strings.Index(source[startIndex:], end)
	if endIndex < 0 {
		t.Fatalf("contract block end %q not found after %q", end, start)
	}
	return source[startIndex : startIndex+endIndex]
}

func normalizeWhitespace(value string) string {
	withoutProtoCommentMarkers := strings.ReplaceAll(value, "//", " ")
	return strings.Join(strings.Fields(withoutProtoCommentMarkers), " ")
}
