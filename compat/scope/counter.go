// Package scope provides executable migration validators for bound app scopes.
package scope

import (
	"fmt"
	"sort"

	counterv1 "github.com/4ubak/cg-proto/gen/go/platform/counter"
)

// CounterRouting contains the canonical bound tuples and the deprecated
// parallel counter routing fields carried during migration.
type CounterRouting struct {
	App             counterv1.CounterApp
	Perspective     counterv1.CounterPerspective
	OrganizationIDs []string
	Scopes          []*counterv1.CounterScope
}

// NormalizeCounterRouting validates canonical counter scopes against legacy
// identity fields. Organization routing is accepted only through a canonical
// tuple whose membership version matches the current source-of-truth version.
func NormalizeCounterRouting(routing CounterRouting, currentVersions map[string]int64) ([]*counterv1.CounterScope, error) {
	canonical, err := validateCounterScopes(routing.Scopes, currentVersions)
	if err != nil {
		return nil, err
	}
	legacy, legacyPresent, err := legacyCounterScopes(routing)
	if err != nil {
		return nil, err
	}
	if len(canonical) > 0 {
		if legacyPresent && !equalCounterIdentities(canonical, legacy) {
			return nil, fmt.Errorf("canonical counter scopes conflict with legacy routing")
		}
		return canonical, nil
	}
	if !legacyPresent {
		return nil, nil
	}
	for _, scope := range legacy {
		if scope.GetOrganizationId() != "" {
			return nil, fmt.Errorf("legacy counter organization routing has no membership_version")
		}
	}
	return validateCounterScopes(legacy, currentVersions)
}

func legacyCounterScopes(routing CounterRouting) ([]*counterv1.CounterScope, bool, error) {
	present := routing.App != counterv1.CounterApp_COUNTER_APP_UNSPECIFIED ||
		routing.Perspective != counterv1.CounterPerspective_COUNTER_PERSPECTIVE_UNSPECIFIED ||
		len(routing.OrganizationIDs) > 0
	if !present {
		return nil, false, nil
	}
	if routing.App == counterv1.CounterApp_COUNTER_APP_UNSPECIFIED ||
		routing.Perspective == counterv1.CounterPerspective_COUNTER_PERSPECTIVE_UNSPECIFIED {
		return nil, true, fmt.Errorf("legacy counter routing is partial")
	}
	organizationIDs := routing.OrganizationIDs
	if len(organizationIDs) == 0 {
		organizationIDs = []string{""}
	}
	scopes := make([]*counterv1.CounterScope, 0, len(organizationIDs))
	seen := make(map[string]struct{}, len(organizationIDs))
	for _, organizationID := range organizationIDs {
		if _, duplicate := seen[organizationID]; duplicate {
			return nil, true, fmt.Errorf("duplicate legacy counter organization_id %q", organizationID)
		}
		seen[organizationID] = struct{}{}
		scopes = append(scopes, &counterv1.CounterScope{
			App:            routing.App,
			Perspective:    routing.Perspective,
			OrganizationId: organizationID,
		})
	}
	return scopes, true, nil
}

func validateCounterScopes(scopes []*counterv1.CounterScope, currentVersions map[string]int64) ([]*counterv1.CounterScope, error) {
	seen := make(map[string]int64, len(scopes))
	result := make([]*counterv1.CounterScope, 0, len(scopes))
	for _, scope := range scopes {
		if scope == nil {
			return nil, fmt.Errorf("counter scope is nil")
		}
		if err := validateCounterScope(scope); err != nil {
			return nil, err
		}
		identity := counterIdentity(scope)
		if version, duplicate := seen[identity]; duplicate {
			if version != scope.GetMembershipVersion() {
				return nil, fmt.Errorf("counter scope %s has conflicting membership versions %d and %d", identity, version, scope.GetMembershipVersion())
			}
			return nil, fmt.Errorf("duplicate counter scope %s", identity)
		}
		seen[identity] = scope.GetMembershipVersion()
		if scope.GetOrganizationId() != "" {
			current, ok := currentVersions[scope.GetOrganizationId()]
			if !ok {
				return nil, fmt.Errorf("current membership version is required for organization %q", scope.GetOrganizationId())
			}
			if current != scope.GetMembershipVersion() {
				return nil, fmt.Errorf("stale counter membership_version %d for organization %q; current is %d", scope.GetMembershipVersion(), scope.GetOrganizationId(), current)
			}
		}
		result = append(result, scope)
	}
	return result, nil
}

func validateCounterScope(scope *counterv1.CounterScope) error {
	if scope.GetApp() != counterv1.CounterApp_COUNTER_APP_CLIENT && scope.GetApp() != counterv1.CounterApp_COUNTER_APP_PRO {
		return fmt.Errorf("counter scope app %s is invalid", scope.GetApp())
	}
	if scope.GetPerspective() == counterv1.CounterPerspective_COUNTER_PERSPECTIVE_UNSPECIFIED {
		return fmt.Errorf("counter scope perspective is required")
	}
	if scope.GetApp() == counterv1.CounterApp_COUNTER_APP_CLIENT && scope.GetPerspective() != counterv1.CounterPerspective_COUNTER_PERSPECTIVE_BUYER {
		return fmt.Errorf("CLIENT counter scope must use BUYER perspective")
	}
	if scope.GetApp() == counterv1.CounterApp_COUNTER_APP_CLIENT && scope.GetOrganizationId() != "" {
		return fmt.Errorf("CLIENT counter scope cannot carry organization_id")
	}
	if scope.GetApp() == counterv1.CounterApp_COUNTER_APP_PRO &&
		scope.GetPerspective() != counterv1.CounterPerspective_COUNTER_PERSPECTIVE_SELLER_ORG &&
		scope.GetPerspective() != counterv1.CounterPerspective_COUNTER_PERSPECTIVE_SELLER_USER &&
		scope.GetPerspective() != counterv1.CounterPerspective_COUNTER_PERSPECTIVE_SUPPORT {
		return fmt.Errorf("PRO counter scope must use a seller or support perspective")
	}
	if scope.GetPerspective() == counterv1.CounterPerspective_COUNTER_PERSPECTIVE_SELLER_ORG && scope.GetOrganizationId() == "" {
		return fmt.Errorf("seller organization counter scope requires organization_id")
	}
	if scope.GetOrganizationId() == "" && scope.GetMembershipVersion() != 0 {
		return fmt.Errorf("membership_version must be 0 without organization_id")
	}
	if scope.GetOrganizationId() != "" && scope.GetMembershipVersion() <= 0 {
		return fmt.Errorf("membership_version must be positive for organization_id")
	}
	return nil
}

func equalCounterIdentities(left, right []*counterv1.CounterScope) bool {
	if len(left) != len(right) {
		return false
	}
	leftKeys := counterIdentityKeys(left)
	rightKeys := counterIdentityKeys(right)
	sort.Strings(leftKeys)
	sort.Strings(rightKeys)
	for i := range leftKeys {
		if leftKeys[i] != rightKeys[i] {
			return false
		}
	}
	return true
}

func counterIdentityKeys(scopes []*counterv1.CounterScope) []string {
	keys := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		keys = append(keys, counterIdentity(scope))
	}
	return keys
}

func counterIdentity(scope *counterv1.CounterScope) string {
	return fmt.Sprintf("%d/%d/%s", scope.GetApp(), scope.GetPerspective(), scope.GetOrganizationId())
}
