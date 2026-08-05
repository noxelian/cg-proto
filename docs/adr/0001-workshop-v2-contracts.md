# ADR 0001: Workshop V2 service contracts

- Status: Accepted for additive contract rollout
- Date: 2026-08-05

## Context and ownership

CRM owns customer communication and sales-pipeline stages. `cg-workshop` owns
inspection, photos, estimates, repair work, vehicle arrival, production status,
operational blockers and its Kanban projection. `cg-payments` owns confirmed
payment and refund facts. BFFs authenticate, authorize, proxy and compose; they
do not coordinate the workflow.

Workshop V2 uses this operational sequence:

`INSPECTION -> WAITING_APPROVAL -> WAITING_PAYMENT -> PAID_WAITING_ARRIVAL -> AcceptVehicle -> QUEUE`

Any confirmed positive partial or full payment may move a V2 order to
`PAID_WAITING_ARRIVAL`. Payment status remains a separate projection from repair
status. `AcceptVehicle` records arrival and publishes the lifecycle fact that CRM
uses to enter its stable `auto_in_repair` system stage. `READY` publishes a fact
that CRM uses to enter `delivery_closing`; it never closes the deal automatically.

## Commands and events

`CreateIntakeOrderFromCRM` is a service-only/replay command for the V2 intake
consumer. Its request is intentionally IDs-only: command key, immutable CRM
source-event id, target workshop id and CRM deal id. It does not accept
`organization_id`, `user_id`, `garage_car_id` or a caller-owned customer/car
snapshot. Before any write, `cg-workshop` resolves `workshop_id` to its
authoritative organization, authorizes the service principal for that tenant,
then loads `crm_deal_id` through CRM under the
resolved organization. Pipeline, stage, customer/user, garage car and display
data come only from that authoritative deal and cg-users linkage. Event
consumers bind `source_event_id` to their durable authenticated inbox envelope;
controlled replay tooling binds it to the stored source event record. Missing
workshop/deal/source records fail `NOT_FOUND`; tenant/authorization mismatches
fail `PERMISSION_DENIED`; unresolved or ambiguous user/car linkage fails
`FAILED_PRECONDITION`. Every path fails closed before mutation. The existing
`CreateOrderFromCRM` RPC remains unchanged for legacy callers.

`AcceptVehicle` is an explicit, idempotent workshop command. The order resolves
the authoritative workshop and tenant, and the authenticated principal is the
arrival audit actor. The request carries no actor or tenant identity.

### Idempotency and conflicts

`CreateIntakeOrderFromCRM.idempotency_key` is scoped to `(authoritative
organization_id, workshop_id, RPC)`. Its semantic fingerprint is the exact tuple
`(source_event_id, workshop_id, crm_deal_id)`; the key itself is excluded. The
scoped key, source event identity `(authenticated CRM producer,
source_event_id)`, and authoritative CRM deal identity `(CRM organization_id,
crm_deal_id)` are independent deduplication identities.

`AcceptVehicle.idempotency_key` is scoped to `(authoritative organization_id,
authoritative workshop_id, RPC)`. Its semantic fingerprint is the exact tuple
`(repair_order_id, note presence, note UTF-8 bytes)`; the key itself is excluded.
The optional `note` is therefore immutable semantic input: absent and empty are
distinct.

For both commands, replaying the same semantics returns the same logical repair
order and sets `already_exists`/`already_accepted` without inserting a second
audit row or publishing a second outbox event. This also applies when another
deduplication identity reaches the same row. Reusing a scoped key, source event
or deal/arrival identity with different semantics returns gRPC `ALREADY_EXISTS`
without mutation. If the supplied deduplication identities resolve to different
rows, the same conflict is returned and the inconsistency is not guessed away.
An already accepted order can never acquire a second arrival audit or event: a
same-fingerprint retry returns the original result, while a different note/order
fingerprint conflicts.

Cross-service mutations are delivered from transactional outboxes and applied by
idempotent inbox consumers. The existing `transaction.succeeded` payment fact is
authoritative for repair-order payment projection. Workshop lifecycle facts
include their event id, repair-order id, workflow, origin and CRM linkage so CRM
can deduplicate them and resolve stages by `Stage.system_code`, never by display
name. Transport retries must not create a second order, arrival audit row,
payment row or CRM transition.

`GetRepairOrderPaymentTarget` is a read-only, service-only contract for
`cg-payments`. It replaces the current two-call read of a broad repair order plus
workshop. Workshop returns authoritative tenant and outstanding amount in whole
tenge; the payments adapter converts it to tiyn. Payment events remain in tiyn,
and the workshop event adapter converts only exact whole-tenge values.

## Authorization model

The stable capability codes are `workshop:intake`, `workshop:operate`,
`workshop:accept_vehicle`, `workshop:record_terminal_payment`,
`payments:confirm_bank_transfer` and `payments:refund`. The `accountant` platform
role is a separate dynamic role definition composed from capabilities. The users
contract already makes `role_codes` authoritative and marks the static
`PlatformRole` enum deprecated, so this ADR intentionally adds no accountant
enum value.

## Compatibility and migration boundary

All protobuf changes are additive: existing field and enum numbers stay intact,
and generated import paths do not change. Unset V2 fields decode to their normal
proto3 defaults, preserving old consumers. `new_order_legacy` orders use the
legacy workflow/origin and remain writable only through the existing legacy
adapter. V2 commands own V2 linkage, intake, arrival and blocker fields. Legacy
and V2 paths must not concurrently update those fields.

`BANK_TRANSFER` is added only to the workshop payment projection. Confirmation
and refund authority remains in `cg-payments`; workshop never manufactures a
financial fact. Consumer releases can therefore be rolled out independently
after a tagged proto release, with legacy traffic continuing throughout.
