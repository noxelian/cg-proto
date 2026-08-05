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
snapshot. `cg-workshop` resolves `workshop_id` to its authoritative organization
and authenticates the service principal before looking at a durable command.
Only a first execution reads source systems. It calls the new CRM-owned
`GetWorkshopIntakeProjection` with the exact organization, workshop, source
event and deal binding. CRM admits only the exact `cg-workshop` service identity,
verifies the four fields against one durable CRM workshop-handoff/outbox record,
loads the deal under that organization, and obtains the minimal user/car fields
through cg-users' exact-`cg-crm` `GetWorkshopIntakeParty`. cg-users
validates that the car belongs to the authoritative deal user and returns no
unrelated profile or garage data. Thus CRM remains the owner of deal linkage,
cg-users remains the owner of user/car facts, and neither human APIs nor BFFs
gain an arbitrary cross-tenant read. Missing workshop/deal/source/user/car
records fail `NOT_FOUND`; caller or binding mismatches fail `PERMISSION_DENIED`;
unresolved or ambiguous linkage fails `FAILED_PRECONDITION`. Every path fails
closed before mutation. The existing `CreateOrderFromCRM` RPC remains unchanged
for legacy callers.

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

The required processing order is normative. `cg-workshop` MUST authenticate and
bind the tenant/service scope before any deduplication lookup. It MUST then look
up the durable command record by the immutable command identities before
reading CRM or cg-users. A matching fingerprint returns the stored result
without re-reading CRM or cg-users; a conflicting fingerprint returns gRPC
`ALREADY_EXISTS` without mutation. Only the first execution may resolve the
current authoritative CRM/user/car sources. After successful resolution, one
local transaction atomically persists the repair order, deduplication records,
intake audit and outbox event. An outage or deletion of a CRM, user or car source
after first success cannot break an exact replay. An exact replay produces no
second intake audit or outbox event.

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

Every successful exact replay under either command that presents a previously
unseen scoped idempotency key MUST bind that key as an alias to the stored
immutable fingerprint and result before returning. The lookup and alias insert
MUST use the same uniqueness and transactional serialization as the existing
key, deal, source-event and arrival identities. If the key is already bound to
any different fingerprint or result, the command returns `ALREADY_EXISTS` with
no mutation. This makes the sequential regression normative: `D1/K1 -> D1/K2
-> D2/K2` first creates result R1, then D1/K2 must bind K2 to R1 before returning
R1, and D2/K2 must then return `ALREADY_EXISTS` without mutation. Under the same
serialization, concurrent D1/K2 and D2/K2 cannot both succeed. Exact replay and
alias binding must not create a second order, intake or arrival audit, outbox
event, or source-system read.

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
