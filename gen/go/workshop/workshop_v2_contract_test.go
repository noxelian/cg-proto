package workshopv1

import (
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestWorkshopV2RepairOrderRoundTrip(t *testing.T) {
	acceptedAt := time.Date(2026, 8, 5, 12, 30, 0, 0, time.UTC)
	original := &RepairOrder{
		Id:       42,
		Status:   RepairStatus_REPAIR_STATUS_PAID_WAITING_ARRIVAL,
		Workflow: RepairOrderWorkflow_REPAIR_ORDER_WORKFLOW_V2,
		Origin:   RepairOrderOrigin_REPAIR_ORDER_ORIGIN_CRM,
		CrmLinkage: &CRMRepairOrderLinkage{
			DealId:                "deal-1",
			PipelineId:            "pipeline-1",
			SourceEventId:         "event-1",
			SourceStageSystemCode: "workshop_intake",
		},
		ArrivalAudit: &VehicleArrivalAudit{
			AcceptedAt:       timestamppb.New(acceptedAt),
			AcceptedByUserId: 77,
			Note:             "front desk",
		},
		ActiveBlockers: []*RepairOrderBlocker{{
			Code:      "vehicle_arrival",
			Type:      RepairOrderBlockerType_REPAIR_ORDER_BLOCKER_TYPE_VEHICLE_ARRIVAL,
			Label:     "Waiting for vehicle",
			BlockedAt: timestamppb.New(acceptedAt.Add(-time.Hour)),
		}},
		ActiveBlockerCount: 1,
	}

	payload, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var decoded RepairOrder
	if err := proto.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got := decoded.GetWorkflow(); got != RepairOrderWorkflow_REPAIR_ORDER_WORKFLOW_V2 {
		t.Fatalf("workflow = %v, want V2", got)
	}
	if got := decoded.GetCrmLinkage().GetSourceEventId(); got != "event-1" {
		t.Fatalf("source_event_id = %q, want event-1", got)
	}
	if got := decoded.GetArrivalAudit().GetAcceptedAt(); got == nil || !got.AsTime().Equal(acceptedAt) {
		t.Fatalf("accepted_at = %v, want %v", got, acceptedAt)
	}
	if got := decoded.GetActiveBlockers()[0].GetType(); got != RepairOrderBlockerType_REPAIR_ORDER_BLOCKER_TYPE_VEHICLE_ARRIVAL {
		t.Fatalf("blocker type = %v, want VEHICLE_ARRIVAL", got)
	}
}

func TestWorkshopV2EnumNumbersRemainAdditive(t *testing.T) {
	if got := int32(RepairStatus_REPAIR_STATUS_CLOSED_BY_CLIENT); got != 13 {
		t.Fatalf("legacy CLOSED_BY_CLIENT number = %d, want 13", got)
	}
	if got := int32(RepairStatus_REPAIR_STATUS_INSPECTION); got != 14 {
		t.Fatalf("INSPECTION number = %d, want 14", got)
	}
	if got := int32(RepairStatus_REPAIR_STATUS_WAITING_PAYMENT); got != 15 {
		t.Fatalf("WAITING_PAYMENT number = %d, want 15", got)
	}
	if got := int32(RepairStatus_REPAIR_STATUS_PAID_WAITING_ARRIVAL); got != 16 {
		t.Fatalf("PAID_WAITING_ARRIVAL number = %d, want 16", got)
	}
	if got := int32(PaymentMethod_PAYMENT_METHOD_BANK_TRANSFER); got != 5 {
		t.Fatalf("BANK_TRANSFER number = %d, want 5", got)
	}
}
