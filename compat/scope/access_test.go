package scope

import (
	"testing"

	bidv1 "github.com/4ubak/cg-proto/gen/go/services/bid"
	requestv1 "github.com/4ubak/cg-proto/gen/go/services/request"
	userv1 "github.com/4ubak/cg-proto/gen/go/users/user"
)

func TestBuyerOrganizationScopesRemainDisjoint(t *testing.T) {
	userBuyer := &userv1.UserAppScope{App: userv1.UserApp_USER_APP_PRO, Perspective: userv1.UserPerspective_USER_PERSPECTIVE_BUYER_ORG, OrganizationId: "org-1", MembershipVersion: 8}
	userSupplier := &userv1.UserAppScope{App: userv1.UserApp_USER_APP_PRO, Perspective: userv1.UserPerspective_USER_PERSPECTIVE_SELLER_ORG, OrganizationId: "org-1", MembershipVersion: 8}
	if err := ValidateUserAppScope(userBuyer, 8); err != nil {
		t.Fatalf("PRO buyer org rejected: %v", err)
	}
	if err := ValidateUserAppScope(userSupplier, 8); err != nil {
		t.Fatalf("PRO supplier org rejected: %v", err)
	}
	if userAppScopeIdentity(userBuyer) == userAppScopeIdentity(userSupplier) {
		t.Fatal("buyer-org and supplier-org user scopes share an identity")
	}

	requestBuyer := &requestv1.RequestAccessScope{App: requestv1.RequestAccessApp_REQUEST_ACCESS_APP_PARTNER, Perspective: requestv1.RequestAccessPerspective_REQUEST_ACCESS_PERSPECTIVE_BUYER_ORG, OrganizationId: "org-1", MembershipVersion: 8}
	requestSupplier := &requestv1.RequestAccessScope{App: requestv1.RequestAccessApp_REQUEST_ACCESS_APP_PARTNER, Perspective: requestv1.RequestAccessPerspective_REQUEST_ACCESS_PERSPECTIVE_SUPPLIER_ORG, OrganizationId: "org-1", MembershipVersion: 8}
	if err := ValidateRequestAccessScope(requestBuyer, 8); err != nil {
		t.Fatalf("PARTNER request buyer org rejected: %v", err)
	}
	if err := ValidateRequestAccessScope(requestSupplier, 8); err != nil {
		t.Fatalf("PARTNER request supplier org rejected: %v", err)
	}
	if requestAccessScopeIdentity(requestBuyer) == requestAccessScopeIdentity(requestSupplier) {
		t.Fatal("buyer-org and supplier-org request scopes share an identity")
	}

	bidBuyer := &bidv1.BidAccessScope{App: bidv1.BidAccessApp_BID_ACCESS_APP_PARTNER, Perspective: bidv1.BidAccessPerspective_BID_ACCESS_PERSPECTIVE_BUYER_ORG, OrganizationId: "org-1", MembershipVersion: 8}
	bidSupplier := &bidv1.BidAccessScope{App: bidv1.BidAccessApp_BID_ACCESS_APP_PARTNER, Perspective: bidv1.BidAccessPerspective_BID_ACCESS_PERSPECTIVE_SUPPLIER_ORG, OrganizationId: "org-1", MembershipVersion: 8}
	if err := ValidateBidAccessScope(bidBuyer, 8); err != nil {
		t.Fatalf("PARTNER bid buyer org rejected: %v", err)
	}
	if err := ValidateBidAccessScope(bidSupplier, 8); err != nil {
		t.Fatalf("PARTNER bid supplier org rejected: %v", err)
	}
	if bidAccessScopeIdentity(bidBuyer) == bidAccessScopeIdentity(bidSupplier) {
		t.Fatal("buyer-org and supplier-org bid scopes share an identity")
	}
}

func TestOrganizationScopeGenerationFailsClosed(t *testing.T) {
	cases := []struct {
		name string
		run  func() error
	}{
		{"user missing generation", func() error {
			return ValidateUserAppScope(&userv1.UserAppScope{App: userv1.UserApp_USER_APP_PRO, Perspective: userv1.UserPerspective_USER_PERSPECTIVE_BUYER_ORG, OrganizationId: "org-1"}, 9)
		}},
		{"request stale generation", func() error {
			return ValidateRequestAccessScope(&requestv1.RequestAccessScope{App: requestv1.RequestAccessApp_REQUEST_ACCESS_APP_PARTNER, Perspective: requestv1.RequestAccessPerspective_REQUEST_ACCESS_PERSPECTIVE_BUYER_ORG, OrganizationId: "org-1", MembershipVersion: 8}, 9)
		}},
		{"bid missing current generation", func() error {
			return ValidateBidAccessScope(&bidv1.BidAccessScope{App: bidv1.BidAccessApp_BID_ACCESS_APP_PARTNER, Perspective: bidv1.BidAccessPerspective_BID_ACCESS_PERSPECTIVE_SUPPLIER_ORG, OrganizationId: "org-1", MembershipVersion: 9}, 0)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); err == nil {
				t.Fatal("invalid or stale organization scope accepted")
			}
		})
	}
}

func TestClientBuyerCannotMasqueradeAsOrganization(t *testing.T) {
	if err := ValidateUserAppScope(&userv1.UserAppScope{App: userv1.UserApp_USER_APP_CLIENT, Perspective: userv1.UserPerspective_USER_PERSPECTIVE_BUYER, OrganizationId: "org-1", MembershipVersion: 1}, 1); err == nil {
		t.Fatal("CLIENT user buyer accepted organization scope")
	}
	if err := ValidateRequestAccessScope(&requestv1.RequestAccessScope{App: requestv1.RequestAccessApp_REQUEST_ACCESS_APP_CLIENT, Perspective: requestv1.RequestAccessPerspective_REQUEST_ACCESS_PERSPECTIVE_BUYER_ORG, OrganizationId: "org-1", MembershipVersion: 1}, 1); err == nil {
		t.Fatal("CLIENT request buyer accepted BUYER_ORG")
	}
	if err := ValidateBidAccessScope(&bidv1.BidAccessScope{App: bidv1.BidAccessApp_BID_ACCESS_APP_CLIENT, Perspective: bidv1.BidAccessPerspective_BID_ACCESS_PERSPECTIVE_BUYER_ORG, OrganizationId: "org-1", MembershipVersion: 1}, 1); err == nil {
		t.Fatal("CLIENT bid buyer accepted BUYER_ORG")
	}
}
