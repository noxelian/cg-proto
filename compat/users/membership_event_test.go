package users

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	organizationv1 "github.com/4ubak/cg-proto/gen/go/users/organization"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestMembershipEnvelopeMatchesCurrentProducerConsumerShapeGolden(t *testing.T) {
	prior := MembershipState{Status: organizationv1.MemberStatus_MEMBER_STATUS_FIRED, MembershipVersion: 1}
	event := membershipEventFixture(
		organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_REHIRE,
		organizationv1.MemberStatus_MEMBER_STATUS_FIRED,
		organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE,
		2,
	)

	encoded, err := MarshalOrganizationMembershipChangedEnvelope(event, prior)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	const golden = `{"id":"membership-event-1","type":"org.member.rehired","source":"organization-service","data":{"event_id":"membership-event-1","organization_id":"org-1","user_id":42,"membership_version":2,"old_status":2,"new_status":1,"event_type":3,"occurred_at":"2026-08-06T01:02:03Z"},"timestamp":"2026-08-06T01:02:03Z","metadata":{"user_id":42}}`
	if string(encoded) != golden {
		t.Fatalf("envelope bytes:\n%s\nwant:\n%s", encoded, golden)
	}

	// Mirrors cg-communication PlatformEvents: it reads type, raw data and
	// metadata.user_id from the shared envelope.
	var consumer struct {
		Type     string          `json:"type"`
		Data     json.RawMessage `json:"data"`
		Metadata struct {
			UserID int64 `json:"user_id"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(encoded, &consumer); err != nil {
		t.Fatalf("decode with current cg-communication shape: %v", err)
	}
	if consumer.Type != OrganizationMemberRehiredEventType || consumer.Metadata.UserID != 42 {
		t.Fatalf("consumer envelope mismatch: %+v", consumer)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(consumer.Data, &payload); err != nil {
		t.Fatalf("decode consumer data: %v", err)
	}
	if got := string(payload["user_id"]); got != "42" {
		t.Fatalf("data.user_id = %s, want numeric 42", got)
	}
	if got := string(payload["membership_version"]); got != "2" {
		t.Fatalf("data.membership_version = %s, want numeric 2", got)
	}
}

func TestMembershipEnvelopeCanonicalRoundTrip(t *testing.T) {
	prior := MembershipState{Status: organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE, MembershipVersion: 7}
	want := membershipEventFixture(
		organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_ROLE_CHANGED,
		organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE,
		organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE,
		7,
	)
	encoded, err := MarshalOrganizationMembershipChangedEnvelope(want, prior)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got organizationv1.OrganizationMembershipChangedEvent
	if err := UnmarshalOrganizationMembershipChangedEnvelope(encoded, &got, prior); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	assertMembershipEventEqual(t, &got, want)
}

func TestMembershipEnvelopeDualReadsCurrentLegacyProducerData(t *testing.T) {
	tests := []struct {
		name        string
		envelope    string
		prior       MembershipState
		wantType    organizationv1.OrganizationMembershipChangeType
		wantOld     organizationv1.MemberStatus
		wantNew     organizationv1.MemberStatus
		wantVersion int64
	}{
		{
			name:        "added",
			envelope:    `{"id":"legacy-add","type":"org.member.added","source":"organization-service","data":{"member_id":"member-1","organization_id":"org-1","user_id":42,"role_code":"master","org_name":"ACME","org_type":"sto"},"timestamp":"2026-08-06T01:02:03Z","metadata":{"user_id":42}}`,
			prior:       MembershipState{},
			wantType:    organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_ADD,
			wantOld:     organizationv1.MemberStatus_MEMBER_STATUS_UNSPECIFIED,
			wantNew:     organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE,
			wantVersion: 1,
		},
		{
			name:        "fired",
			envelope:    `{"id":"legacy-fire","type":"org.member.fired","source":"organization-service","data":{"member_id":"member-1","organization_id":"org-1","user_id":42,"role_code":"master","org_name":"ACME","reason":"left"},"timestamp":"2026-08-06T01:02:03Z","metadata":{"user_id":42}}`,
			prior:       MembershipState{Status: organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE, MembershipVersion: 4},
			wantType:    organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_FIRE,
			wantOld:     organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE,
			wantNew:     organizationv1.MemberStatus_MEMBER_STATUS_FIRED,
			wantVersion: 4,
		},
		{
			name:        "rehired",
			envelope:    `{"id":"legacy-rehire","type":"org.member.rehired","source":"organization-service","data":{"member_id":"member-1","organization_id":"org-1","user_id":42,"role_code":"master","org_name":"ACME","previous_fired_at":"2026-08-01T00:00:00Z"},"timestamp":"2026-08-06T01:02:03Z","metadata":{"user_id":42}}`,
			prior:       MembershipState{Status: organizationv1.MemberStatus_MEMBER_STATUS_FIRED, MembershipVersion: 4},
			wantType:    organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_REHIRE,
			wantOld:     organizationv1.MemberStatus_MEMBER_STATUS_FIRED,
			wantNew:     organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE,
			wantVersion: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got organizationv1.OrganizationMembershipChangedEvent
			if err := UnmarshalOrganizationMembershipChangedEnvelope([]byte(tt.envelope), &got, tt.prior); err != nil {
				t.Fatalf("dual-read legacy envelope: %v", err)
			}
			if got.GetEventType() != tt.wantType || got.GetOldStatus() != tt.wantOld ||
				got.GetNewStatus() != tt.wantNew || got.GetMembershipVersion() != tt.wantVersion {
				t.Fatalf("decoded transition = %+v", &got)
			}
		})
	}
}

func TestMembershipEnvelopeJSONCompatibilityAndStrictness(t *testing.T) {
	prior := MembershipState{Status: organizationv1.MemberStatus_MEMBER_STATUS_FIRED, MembershipVersion: 1}
	base := `{"id":"membership-event-1","type":"org.member.rehired","source":"organization-service","data":{"event_id":"membership-event-1","organization_id":"org-1","user_id":42,"membership_version":2,"old_status":2,"new_status":1,"event_type":3,"occurred_at":"2026-08-06T01:02:03Z"},"timestamp":"2026-08-06T01:02:03Z","metadata":{"user_id":42}}`

	t.Run("additive unknown fields accepted", func(t *testing.T) {
		withUnknown := strings.Replace(base, `"data":{`, `"future_envelope_field":{"v":1},"data":{"future_payload_field":true,`, 1)
		var got organizationv1.OrganizationMembershipChangedEvent
		if err := UnmarshalOrganizationMembershipChangedEnvelope([]byte(withUnknown), &got, prior); err != nil {
			t.Fatalf("unknown additive fields: %v", err)
		}
	})

	t.Run("trailing second JSON rejected", func(t *testing.T) {
		var got organizationv1.OrganizationMembershipChangedEvent
		if err := UnmarshalOrganizationMembershipChangedEnvelope([]byte(base+` {}`), &got, prior); err == nil {
			t.Fatal("expected trailing JSON rejection")
		}
	})

	t.Run("quoted legacy int64 rejected", func(t *testing.T) {
		quoted := strings.Replace(base, `"user_id":42`, `"user_id":"42"`, 1)
		var got organizationv1.OrganizationMembershipChangedEvent
		if err := UnmarshalOrganizationMembershipChangedEnvelope([]byte(quoted), &got, prior); err == nil {
			t.Fatal("expected quoted int64 rejection")
		}
	})
}

func TestMembershipEnvelopeRejectsCrossLayerIdentityMismatch(t *testing.T) {
	prior := MembershipState{Status: organizationv1.MemberStatus_MEMBER_STATUS_FIRED, MembershipVersion: 1}
	base := `{"id":"membership-event-1","type":"org.member.rehired","source":"organization-service","data":{"event_id":"membership-event-1","organization_id":"org-1","user_id":42,"membership_version":2,"old_status":2,"new_status":1,"event_type":3,"occurred_at":"2026-08-06T01:02:03Z"},"timestamp":"2026-08-06T01:02:03Z","metadata":{"user_id":42}}`
	tests := map[string]string{
		"event id":         strings.Replace(base, `"event_id":"membership-event-1"`, `"event_id":"other"`, 1),
		"event type":       strings.Replace(base, `"event_type":3`, `"event_type":2`, 1),
		"metadata user":    strings.Replace(base, `"metadata":{"user_id":42}`, `"metadata":{"user_id":43}`, 1),
		"timestamp":        strings.Replace(base, `"occurred_at":"2026-08-06T01:02:03Z"`, `"occurred_at":"2026-08-07T01:02:03Z"`, 1),
		"missing id":       strings.Replace(base, `"id":"membership-event-1"`, `"id":""`, 1),
		"missing type":     strings.Replace(base, `"type":"org.member.rehired"`, `"type":""`, 1),
		"wrong type":       strings.Replace(base, `"type":"org.member.rehired"`, `"type":"org.member.unknown"`, 1),
		"missing source":   strings.Replace(base, `"source":"organization-service"`, `"source":""`, 1),
		"wrong source":     strings.Replace(base, `"source":"organization-service"`, `"source":"other"`, 1),
		"missing metadata": strings.Replace(base, `"metadata":{"user_id":42}`, `"metadata":{}`, 1),
	}
	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			var got organizationv1.OrganizationMembershipChangedEvent
			if err := UnmarshalOrganizationMembershipChangedEnvelope([]byte(raw), &got, prior); err == nil {
				t.Fatal("expected cross-layer mismatch rejection")
			}
		})
	}
}

func TestValidateOrganizationMembershipTransition(t *testing.T) {
	active4 := MembershipState{Status: organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE, MembershipVersion: 4}
	fired4 := MembershipState{Status: organizationv1.MemberStatus_MEMBER_STATUS_FIRED, MembershipVersion: 4}
	tests := []struct {
		name    string
		event   *organizationv1.OrganizationMembershipChangedEvent
		prior   MembershipState
		wantErr bool
	}{
		{"valid add", membershipEventFixture(organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_ADD, organizationv1.MemberStatus_MEMBER_STATUS_UNSPECIFIED, organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE, 1), MembershipState{}, false},
		{"valid fire", membershipEventFixture(organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_FIRE, organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE, organizationv1.MemberStatus_MEMBER_STATUS_FIRED, 4), active4, false},
		{"valid rehire", membershipEventFixture(organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_REHIRE, organizationv1.MemberStatus_MEMBER_STATUS_FIRED, organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE, 5), fired4, false},
		{"valid role changed", membershipEventFixture(organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_ROLE_CHANGED, organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE, organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE, 4), active4, false},
		{"nil event", nil, MembershipState{}, true},
		{"empty event id", mutateMembershipEvent(membershipEventFixture(organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_ADD, organizationv1.MemberStatus_MEMBER_STATUS_UNSPECIFIED, organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE, 1), func(event *organizationv1.OrganizationMembershipChangedEvent) { event.EventId = "" }), MembershipState{}, true},
		{"empty organization id", mutateMembershipEvent(membershipEventFixture(organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_ADD, organizationv1.MemberStatus_MEMBER_STATUS_UNSPECIFIED, organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE, 1), func(event *organizationv1.OrganizationMembershipChangedEvent) { event.OrganizationId = "" }), MembershipState{}, true},
		{"non-positive user", mutateMembershipEvent(membershipEventFixture(organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_ADD, organizationv1.MemberStatus_MEMBER_STATUS_UNSPECIFIED, organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE, 1), func(event *organizationv1.OrganizationMembershipChangedEvent) { event.UserId = 0 }), MembershipState{}, true},
		{"non-positive version", mutateMembershipEvent(membershipEventFixture(organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_ADD, organizationv1.MemberStatus_MEMBER_STATUS_UNSPECIFIED, organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE, 1), func(event *organizationv1.OrganizationMembershipChangedEvent) { event.MembershipVersion = 0 }), MembershipState{}, true},
		{"missing timestamp", mutateMembershipEvent(membershipEventFixture(organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_ADD, organizationv1.MemberStatus_MEMBER_STATUS_UNSPECIFIED, organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE, 1), func(event *organizationv1.OrganizationMembershipChangedEvent) { event.OccurredAt = nil }), MembershipState{}, true},
		{"invalid timestamp", mutateMembershipEvent(membershipEventFixture(organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_ADD, organizationv1.MemberStatus_MEMBER_STATUS_UNSPECIFIED, organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE, 1), func(event *organizationv1.OrganizationMembershipChangedEvent) {
			event.OccurredAt = &timestamppb.Timestamp{Seconds: 253402300800}
		}), MembershipState{}, true},
		{"add wrong version", membershipEventFixture(organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_ADD, organizationv1.MemberStatus_MEMBER_STATUS_UNSPECIFIED, organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE, 2), MembershipState{}, true},
		{"add from prior member", membershipEventFixture(organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_ADD, organizationv1.MemberStatus_MEMBER_STATUS_UNSPECIFIED, organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE, 1), active4, true},
		{"fire increments", membershipEventFixture(organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_FIRE, organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE, organizationv1.MemberStatus_MEMBER_STATUS_FIRED, 5), active4, true},
		{"fire impossible statuses", membershipEventFixture(organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_FIRE, organizationv1.MemberStatus_MEMBER_STATUS_FIRED, organizationv1.MemberStatus_MEMBER_STATUS_FIRED, 4), active4, true},
		{"rehire does not increment", membershipEventFixture(organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_REHIRE, organizationv1.MemberStatus_MEMBER_STATUS_FIRED, organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE, 4), fired4, true},
		{"role changed changes status", membershipEventFixture(organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_ROLE_CHANGED, organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE, organizationv1.MemberStatus_MEMBER_STATUS_FIRED, 4), active4, true},
		{"unspecified type", membershipEventFixture(organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_UNSPECIFIED, organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE, organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE, 4), active4, true},
		{"invalid type enum", membershipEventFixture(organizationv1.OrganizationMembershipChangeType(99), organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE, organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE, 4), active4, true},
		{"invalid prior status", membershipEventFixture(organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_FIRE, organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE, organizationv1.MemberStatus_MEMBER_STATUS_FIRED, 4), MembershipState{Status: organizationv1.MemberStatus(99), MembershipVersion: 4}, true},
		{"unspecified prior with version", membershipEventFixture(organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_ADD, organizationv1.MemberStatus_MEMBER_STATUS_UNSPECIFIED, organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE, 1), MembershipState{MembershipVersion: 1}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOrganizationMembershipTransition(tt.event, tt.prior)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateOrganizationMembershipTransition() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMembershipEnvelopeMarshalAndUnmarshalInvokeLifecycleValidator(t *testing.T) {
	invalid := membershipEventFixture(
		organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_REHIRE,
		organizationv1.MemberStatus_MEMBER_STATUS_FIRED,
		organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE,
		1,
	)
	prior := MembershipState{Status: organizationv1.MemberStatus_MEMBER_STATUS_FIRED, MembershipVersion: 1}
	if _, err := MarshalOrganizationMembershipChangedEnvelope(invalid, prior); err == nil {
		t.Fatal("marshal accepted invalid lifecycle")
	}
	legacy := `{"id":"legacy-rehire","type":"org.member.rehired","source":"organization-service","data":{"organization_id":"org-1","user_id":42},"timestamp":"2026-08-06T01:02:03Z","metadata":{"user_id":42}}`
	var decoded organizationv1.OrganizationMembershipChangedEvent
	if err := UnmarshalOrganizationMembershipChangedEnvelope([]byte(legacy), &decoded, MembershipState{}); err == nil {
		t.Fatal("unmarshal accepted invalid prior lifecycle")
	}
}

func membershipEventFixture(changeType organizationv1.OrganizationMembershipChangeType, oldStatus, newStatus organizationv1.MemberStatus, version int64) *organizationv1.OrganizationMembershipChangedEvent {
	return &organizationv1.OrganizationMembershipChangedEvent{
		EventId:           "membership-event-1",
		OrganizationId:    "org-1",
		UserId:            42,
		MembershipVersion: version,
		OldStatus:         oldStatus,
		NewStatus:         newStatus,
		EventType:         changeType,
		OccurredAt:        timestamppb.New(time.Date(2026, time.August, 6, 1, 2, 3, 0, time.UTC)),
	}
}

func mutateMembershipEvent(event *organizationv1.OrganizationMembershipChangedEvent, mutate func(*organizationv1.OrganizationMembershipChangedEvent)) *organizationv1.OrganizationMembershipChangedEvent {
	mutate(event)
	return event
}

func assertMembershipEventEqual(t *testing.T, got, want *organizationv1.OrganizationMembershipChangedEvent) {
	t.Helper()
	if got.GetEventId() != want.GetEventId() || got.GetOrganizationId() != want.GetOrganizationId() ||
		got.GetUserId() != want.GetUserId() || got.GetMembershipVersion() != want.GetMembershipVersion() ||
		got.GetOldStatus() != want.GetOldStatus() || got.GetNewStatus() != want.GetNewStatus() ||
		got.GetEventType() != want.GetEventType() ||
		!got.GetOccurredAt().AsTime().Equal(want.GetOccurredAt().AsTime()) {
		t.Fatalf("event mismatch: got %+v, want %+v", got, want)
	}
}
