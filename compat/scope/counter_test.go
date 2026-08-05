package scope

import (
	"testing"

	counterv1 "github.com/4ubak/cg-proto/gen/go/platform/counter"
)

func TestNormalizeCounterRouting(t *testing.T) {
	org1 := counterOrganizationScope("org-1", 7)
	org2 := counterOrganizationScope("org-2", 12)

	t.Run("matching multiple bound tuples preserve each generation", func(t *testing.T) {
		got, err := NormalizeCounterRouting(CounterRouting{
			App:             counterv1.CounterApp_COUNTER_APP_PRO,
			Perspective:     counterv1.CounterPerspective_COUNTER_PERSPECTIVE_SELLER_ORG,
			OrganizationIDs: []string{"org-1", "org-2"},
			Scopes:          []*counterv1.CounterScope{org1, org2},
		}, map[string]int64{"org-1": 7, "org-2": 12})
		if err != nil {
			t.Fatalf("normalize: %v", err)
		}
		if len(got) != 2 || got[0] != org1 || got[1] != org2 {
			t.Fatalf("normalized scopes = %+v", got)
		}
	})

	t.Run("legacy identity conflict is rejected", func(t *testing.T) {
		_, err := NormalizeCounterRouting(CounterRouting{
			App:             counterv1.CounterApp_COUNTER_APP_PRO,
			Perspective:     counterv1.CounterPerspective_COUNTER_PERSPECTIVE_SELLER_ORG,
			OrganizationIDs: []string{"org-other"},
			Scopes:          []*counterv1.CounterScope{org1},
		}, map[string]int64{"org-1": 7})
		if err == nil {
			t.Fatal("expected legacy identity conflict")
		}
	})

	t.Run("same identity with different generations is rejected", func(t *testing.T) {
		_, err := NormalizeCounterRouting(CounterRouting{
			Scopes: []*counterv1.CounterScope{org1, counterOrganizationScope("org-1", 8)},
		}, map[string]int64{"org-1": 8})
		if err == nil {
			t.Fatal("expected generation conflict")
		}
	})

	t.Run("stale generation is rejected", func(t *testing.T) {
		_, err := NormalizeCounterRouting(CounterRouting{Scopes: []*counterv1.CounterScope{org1}}, map[string]int64{"org-1": 8})
		if err == nil {
			t.Fatal("expected stale generation rejection")
		}
	})

	t.Run("legacy organization routing cannot invent a generation", func(t *testing.T) {
		_, err := NormalizeCounterRouting(CounterRouting{
			App:             counterv1.CounterApp_COUNTER_APP_PRO,
			Perspective:     counterv1.CounterPerspective_COUNTER_PERSPECTIVE_SELLER_ORG,
			OrganizationIDs: []string{"org-1"},
		}, map[string]int64{"org-1": 7})
		if err == nil {
			t.Fatal("expected missing membership_version rejection")
		}
	})

	t.Run("non organization legacy migration uses version zero", func(t *testing.T) {
		got, err := NormalizeCounterRouting(CounterRouting{
			App:         counterv1.CounterApp_COUNTER_APP_CLIENT,
			Perspective: counterv1.CounterPerspective_COUNTER_PERSPECTIVE_BUYER,
		}, nil)
		if err != nil {
			t.Fatalf("normalize: %v", err)
		}
		if len(got) != 1 || got[0].GetOrganizationId() != "" || got[0].GetMembershipVersion() != 0 {
			t.Fatalf("non-org scope = %+v", got)
		}
	})

	t.Run("non organization canonical scope rejects a detached generation", func(t *testing.T) {
		_, err := NormalizeCounterRouting(CounterRouting{Scopes: []*counterv1.CounterScope{{
			App:               counterv1.CounterApp_COUNTER_APP_CLIENT,
			Perspective:       counterv1.CounterPerspective_COUNTER_PERSPECTIVE_BUYER,
			MembershipVersion: 1,
		}}}, nil)
		if err == nil {
			t.Fatal("expected non-org version rejection")
		}
	})
}

func counterOrganizationScope(organizationID string, version int64) *counterv1.CounterScope {
	return &counterv1.CounterScope{
		App:               counterv1.CounterApp_COUNTER_APP_PRO,
		Perspective:       counterv1.CounterPerspective_COUNTER_PERSPECTIVE_SELLER_ORG,
		OrganizationId:    organizationID,
		MembershipVersion: version,
	}
}
