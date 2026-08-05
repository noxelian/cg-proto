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
consumer. Its idempotency key and CRM deal linkage are stable deduplication
identities. The existing `CreateOrderFromCRM` RPC remains unchanged for legacy
callers. `AcceptVehicle` is an explicit, idempotent workshop command; the actor is
bound from authenticated context, not trusted from request data.

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
