// Package users preserves the cg-shared-libs Kafka envelope used by current
// cg-users producers and cg-communication platform-event consumers while the
// membership payload itself is owned by protobuf.
package users

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	organizationv1 "github.com/4ubak/cg-proto/gen/go/users/organization"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	OrganizationMembershipEventSource      = "organization-service"
	OrganizationMemberAddedEventType       = "org.member.added"
	OrganizationMemberFiredEventType       = "org.member.fired"
	OrganizationMemberRehiredEventType     = "org.member.rehired"
	OrganizationMemberRoleChangedEventType = "org.member.role_changed"
)

var errNilMembershipEvent = errors.New("organization membership changed event is nil")

// MembershipState is the organization owner's state immediately before an
// event. Producers and consumers MUST load this authoritative state and pass it
// to both validation and envelope encoding/decoding; JWT/routing state is not a
// substitute. UNSPECIFIED is valid only with version zero.
type MembershipState struct {
	Status            organizationv1.MemberStatus
	MembershipVersion int64
}

// membershipChangedData is the protobuf-owned payload nested inside the exact
// cg-shared-libs kafka.Event data field. Int64 values intentionally stay JSON
// numbers for the encoding/json consumers already in production.
type membershipChangedData struct {
	EventID           string                                           `json:"event_id,omitempty"`
	OrganizationID    string                                           `json:"organization_id"`
	UserID            int64                                            `json:"user_id"`
	MembershipVersion *int64                                           `json:"membership_version,omitempty"`
	OldStatus         *organizationv1.MemberStatus                     `json:"old_status,omitempty"`
	NewStatus         *organizationv1.MemberStatus                     `json:"new_status,omitempty"`
	EventType         *organizationv1.OrganizationMembershipChangeType `json:"event_type,omitempty"`
	OccurredAt        string                                           `json:"occurred_at,omitempty"`
}

// organizationMembershipEventEnvelope mirrors cg-shared-libs/kafka.Event.
// Field order is intentional and covered by a golden test because existing
// cg-users publishes this envelope and cg-communication decodes type, data and
// metadata.user_id from it.
type organizationMembershipEventEnvelope struct {
	ID        string                              `json:"id"`
	Type      string                              `json:"type"`
	Source    string                              `json:"source"`
	Data      json.RawMessage                     `json:"data"`
	Timestamp string                              `json:"timestamp"`
	Metadata  organizationMembershipEventMetadata `json:"metadata,omitempty"`
}

type organizationMembershipEventMetadata struct {
	UserID int64 `json:"user_id,omitempty"`
}

// ValidateOrganizationMembershipTransition is the mandatory owner-boundary
// lifecycle validator. It proves the event against the state immediately
// before the transition, including generation increments/retention.
func ValidateOrganizationMembershipTransition(event *organizationv1.OrganizationMembershipChangedEvent, prior MembershipState) error {
	if event == nil {
		return errNilMembershipEvent
	}
	if strings.TrimSpace(event.GetEventId()) == "" {
		return errors.New("membership event_id is required")
	}
	if strings.TrimSpace(event.GetOrganizationId()) == "" {
		return errors.New("membership organization_id is required")
	}
	if event.GetUserId() <= 0 {
		return errors.New("membership user_id must be positive")
	}
	if event.GetMembershipVersion() <= 0 {
		return errors.New("membership_version must be positive")
	}
	if event.GetOccurredAt() == nil {
		return errors.New("membership occurred_at is required")
	}
	if err := event.GetOccurredAt().CheckValid(); err != nil || event.GetOccurredAt().AsTime().IsZero() {
		return errors.New("membership occurred_at is invalid")
	}
	if err := validatePriorMembershipState(prior); err != nil {
		return err
	}

	switch event.GetEventType() {
	case organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_ADD:
		if prior.Status != organizationv1.MemberStatus_MEMBER_STATUS_UNSPECIFIED || prior.MembershipVersion != 0 ||
			event.GetOldStatus() != organizationv1.MemberStatus_MEMBER_STATUS_UNSPECIFIED ||
			event.GetNewStatus() != organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE ||
			event.GetMembershipVersion() != 1 {
			return errors.New("ADD must be UNSPECIFIED/0 -> ACTIVE/1")
		}
	case organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_FIRE:
		if prior.Status != organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE ||
			event.GetOldStatus() != organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE ||
			event.GetNewStatus() != organizationv1.MemberStatus_MEMBER_STATUS_FIRED ||
			event.GetMembershipVersion() != prior.MembershipVersion {
			return errors.New("FIRE must be ACTIVE -> FIRED in the current generation")
		}
	case organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_REHIRE:
		if prior.Status != organizationv1.MemberStatus_MEMBER_STATUS_FIRED ||
			event.GetOldStatus() != organizationv1.MemberStatus_MEMBER_STATUS_FIRED ||
			event.GetNewStatus() != organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE ||
			event.GetMembershipVersion() != prior.MembershipVersion+1 {
			return errors.New("REHIRE must be FIRED -> ACTIVE in the next generation")
		}
	case organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_ROLE_CHANGED:
		if prior.Status != organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE ||
			event.GetOldStatus() != organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE ||
			event.GetNewStatus() != organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE ||
			event.GetMembershipVersion() != prior.MembershipVersion {
			return errors.New("ROLE_CHANGED must remain ACTIVE in the current generation")
		}
	default:
		return errors.New("membership event_type is unspecified or invalid")
	}
	return nil
}

// MarshalOrganizationMembershipChangedEnvelope emits the exact shared
// {id,type,source,data,timestamp,metadata} envelope. Validation against prior
// authoritative state is mandatory and happens before any bytes are returned.
func MarshalOrganizationMembershipChangedEnvelope(event *organizationv1.OrganizationMembershipChangedEvent, prior MembershipState) ([]byte, error) {
	if err := ValidateOrganizationMembershipTransition(event, prior); err != nil {
		return nil, err
	}
	eventType, err := membershipEventTypeName(event.GetEventType())
	if err != nil {
		return nil, err
	}
	version := event.GetMembershipVersion()
	oldStatus := event.GetOldStatus()
	newStatus := event.GetNewStatus()
	changeType := event.GetEventType()
	data, err := json.Marshal(membershipChangedData{
		EventID:           event.GetEventId(),
		OrganizationID:    event.GetOrganizationId(),
		UserID:            event.GetUserId(),
		MembershipVersion: &version,
		OldStatus:         &oldStatus,
		NewStatus:         &newStatus,
		EventType:         &changeType,
		OccurredAt:        event.GetOccurredAt().AsTime().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return nil, fmt.Errorf("marshal membership event data: %w", err)
	}
	return json.Marshal(organizationMembershipEventEnvelope{
		ID:        event.GetEventId(),
		Type:      eventType,
		Source:    OrganizationMembershipEventSource,
		Data:      data,
		Timestamp: event.GetOccurredAt().AsTime().UTC().Format(time.RFC3339),
		Metadata: organizationMembershipEventMetadata{
			UserID: event.GetUserId(),
		},
	})
}

// UnmarshalOrganizationMembershipChangedEnvelope dual-reads the current
// cg-users legacy data shape and the canonical generation payload. Additive
// unknown fields are accepted. The outer and nested objects are decoded with
// json.Unmarshal, which rejects trailing/multiple JSON values and quoted int64
// values while keeping current encoding/json compatibility.
func UnmarshalOrganizationMembershipChangedEnvelope(data []byte, event *organizationv1.OrganizationMembershipChangedEvent, prior MembershipState) error {
	if event == nil {
		return errNilMembershipEvent
	}
	var envelope organizationMembershipEventEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return fmt.Errorf("unmarshal membership event envelope: %w", err)
	}
	if strings.TrimSpace(envelope.ID) == "" || strings.TrimSpace(envelope.Type) == "" || strings.TrimSpace(envelope.Source) == "" {
		return errors.New("membership envelope id, type and source are required")
	}
	if envelope.Source != OrganizationMembershipEventSource {
		return fmt.Errorf("membership envelope source %q is invalid", envelope.Source)
	}
	changeType, err := membershipChangeType(envelope.Type)
	if err != nil {
		return err
	}
	occurredAt, err := time.Parse(time.RFC3339Nano, envelope.Timestamp)
	if err != nil {
		return fmt.Errorf("membership envelope timestamp is invalid: %w", err)
	}
	if envelope.Metadata.UserID <= 0 {
		return errors.New("membership envelope metadata.user_id must be positive")
	}
	var payload membershipChangedData
	if err := json.Unmarshal(envelope.Data, &payload); err != nil {
		return fmt.Errorf("unmarshal membership event data: %w", err)
	}
	if payload.OrganizationID == "" || payload.UserID <= 0 {
		return errors.New("membership event data organization_id and positive user_id are required")
	}
	if payload.UserID != envelope.Metadata.UserID {
		return errors.New("membership event data user_id conflicts with metadata.user_id")
	}

	decoded := &organizationv1.OrganizationMembershipChangedEvent{
		EventId:        envelope.ID,
		OrganizationId: payload.OrganizationID,
		UserId:         payload.UserID,
		EventType:      changeType,
		OccurredAt:     timestamppb.New(occurredAt),
	}
	canonical := payload.EventID != "" || payload.MembershipVersion != nil || payload.OldStatus != nil ||
		payload.NewStatus != nil || payload.EventType != nil || payload.OccurredAt != ""
	if canonical {
		if payload.EventID == "" || payload.MembershipVersion == nil || payload.OldStatus == nil ||
			payload.NewStatus == nil || payload.EventType == nil || payload.OccurredAt == "" {
			return errors.New("canonical membership event data is partial")
		}
		if payload.EventID != envelope.ID {
			return errors.New("membership event data event_id conflicts with envelope id")
		}
		if *payload.EventType != changeType {
			return errors.New("membership event data event_type conflicts with envelope type")
		}
		payloadTime, err := time.Parse(time.RFC3339Nano, payload.OccurredAt)
		if err != nil {
			return fmt.Errorf("membership event data occurred_at is invalid: %w", err)
		}
		if !payloadTime.UTC().Truncate(time.Second).Equal(occurredAt.UTC().Truncate(time.Second)) {
			return errors.New("membership event data occurred_at conflicts with envelope timestamp")
		}
		decoded.MembershipVersion = *payload.MembershipVersion
		decoded.OldStatus = *payload.OldStatus
		decoded.NewStatus = *payload.NewStatus
		decoded.OccurredAt = timestamppb.New(payloadTime)
	} else {
		applyLegacyTransition(decoded, prior)
	}
	if err := ValidateOrganizationMembershipTransition(decoded, prior); err != nil {
		return err
	}
	event.Reset()
	event.EventId = decoded.GetEventId()
	event.OrganizationId = decoded.GetOrganizationId()
	event.UserId = decoded.GetUserId()
	event.MembershipVersion = decoded.GetMembershipVersion()
	event.OldStatus = decoded.GetOldStatus()
	event.NewStatus = decoded.GetNewStatus()
	event.EventType = decoded.GetEventType()
	event.OccurredAt = decoded.GetOccurredAt()
	return nil
}

func validatePriorMembershipState(prior MembershipState) error {
	switch prior.Status {
	case organizationv1.MemberStatus_MEMBER_STATUS_UNSPECIFIED:
		if prior.MembershipVersion != 0 {
			return errors.New("prior UNSPECIFIED membership must have version zero")
		}
	case organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE, organizationv1.MemberStatus_MEMBER_STATUS_FIRED:
		if prior.MembershipVersion <= 0 {
			return errors.New("prior active/fired membership must have a positive version")
		}
	default:
		return errors.New("prior membership status is invalid")
	}
	return nil
}

func applyLegacyTransition(event *organizationv1.OrganizationMembershipChangedEvent, prior MembershipState) {
	switch event.GetEventType() {
	case organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_ADD:
		event.MembershipVersion = 1
		event.OldStatus = organizationv1.MemberStatus_MEMBER_STATUS_UNSPECIFIED
		event.NewStatus = organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE
	case organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_FIRE:
		event.MembershipVersion = prior.MembershipVersion
		event.OldStatus = organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE
		event.NewStatus = organizationv1.MemberStatus_MEMBER_STATUS_FIRED
	case organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_REHIRE:
		event.MembershipVersion = prior.MembershipVersion + 1
		event.OldStatus = organizationv1.MemberStatus_MEMBER_STATUS_FIRED
		event.NewStatus = organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE
	case organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_ROLE_CHANGED:
		event.MembershipVersion = prior.MembershipVersion
		event.OldStatus = organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE
		event.NewStatus = organizationv1.MemberStatus_MEMBER_STATUS_ACTIVE
	}
}

func membershipEventTypeName(changeType organizationv1.OrganizationMembershipChangeType) (string, error) {
	switch changeType {
	case organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_ADD:
		return OrganizationMemberAddedEventType, nil
	case organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_FIRE:
		return OrganizationMemberFiredEventType, nil
	case organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_REHIRE:
		return OrganizationMemberRehiredEventType, nil
	case organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_ROLE_CHANGED:
		return OrganizationMemberRoleChangedEventType, nil
	default:
		return "", errors.New("membership event_type is unspecified or invalid")
	}
}

func membershipChangeType(eventType string) (organizationv1.OrganizationMembershipChangeType, error) {
	switch eventType {
	case OrganizationMemberAddedEventType:
		return organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_ADD, nil
	case OrganizationMemberFiredEventType:
		return organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_FIRE, nil
	case OrganizationMemberRehiredEventType:
		return organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_REHIRE, nil
	case OrganizationMemberRoleChangedEventType:
		return organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_ROLE_CHANGED, nil
	default:
		return organizationv1.OrganizationMembershipChangeType_ORGANIZATION_MEMBERSHIP_CHANGE_TYPE_UNSPECIFIED,
			fmt.Errorf("membership envelope type %q is invalid", eventType)
	}
}
