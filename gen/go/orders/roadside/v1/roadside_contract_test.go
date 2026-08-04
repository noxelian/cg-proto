package roadsidev1

import (
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRoadsideAppointmentFieldsRoundTrip(t *testing.T) {
	appointment := time.Date(2026, 8, 5, 9, 0, 0, 0, time.UTC)
	original := &AssignRoadsideOrderRequest{
		OrderId: "order-1",
		Assignee: &AssignRoadsideOrderRequest_Organization{
			Organization: &OrganizationRoadsideAssignee{
				OrganizationId: "org-1",
				AppointmentAt:  timestamppb.New(appointment),
			},
		},
	}

	payload, err := proto.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var decoded AssignRoadsideOrderRequest
	if err := proto.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got := decoded.GetOrganization().GetAppointmentAt(); got == nil || !got.AsTime().Equal(appointment) {
		t.Fatalf("appointment_at = %v, want %v", got, appointment)
	}

	assignment := &RoadsideAssignment{AppointmentAt: timestamppb.New(appointment)}
	payload, err = proto.Marshal(assignment)
	if err != nil {
		t.Fatalf("Marshal assignment: %v", err)
	}
	var decodedAssignment RoadsideAssignment
	if err := proto.Unmarshal(payload, &decodedAssignment); err != nil {
		t.Fatalf("Unmarshal assignment: %v", err)
	}
	if got := decodedAssignment.GetAppointmentAt(); got == nil || !got.AsTime().Equal(appointment) {
		t.Fatalf("assignment appointment_at = %v, want %v", got, appointment)
	}
}
