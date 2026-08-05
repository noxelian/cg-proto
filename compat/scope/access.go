package scope

import (
	"fmt"

	bidv1 "github.com/4ubak/cg-proto/gen/go/services/bid"
	requestv1 "github.com/4ubak/cg-proto/gen/go/services/request"
	userv1 "github.com/4ubak/cg-proto/gen/go/users/user"
)

// These validators cover the contract-level shape of an authenticated scope.
// Owners must still bind the verified user identity from auth and compare the
// organization membership_version with the current source-of-truth record.

func ValidateUserAppScope(value *userv1.UserAppScope, currentMembershipVersion int64) error {
	if value == nil {
		return fmt.Errorf("user app scope is required")
	}
	switch value.GetApp() {
	case userv1.UserApp_USER_APP_CLIENT:
		if value.GetPerspective() != userv1.UserPerspective_USER_PERSPECTIVE_BUYER {
			return fmt.Errorf("CLIENT user scope must use BUYER perspective")
		}
		if value.GetOrganizationId() != "" {
			return fmt.Errorf("CLIENT buyer user scope cannot carry organization_id")
		}
	case userv1.UserApp_USER_APP_PRO:
		switch value.GetPerspective() {
		case userv1.UserPerspective_USER_PERSPECTIVE_BUYER_ORG,
			userv1.UserPerspective_USER_PERSPECTIVE_SELLER_ORG,
			userv1.UserPerspective_USER_PERSPECTIVE_SELLER_USER:
		default:
			return fmt.Errorf("PRO user scope must use BUYER_ORG or a seller perspective")
		}
	default:
		return fmt.Errorf("user scope app %s is invalid", value.GetApp())
	}
	if (value.GetPerspective() == userv1.UserPerspective_USER_PERSPECTIVE_BUYER_ORG ||
		value.GetPerspective() == userv1.UserPerspective_USER_PERSPECTIVE_SELLER_ORG) && value.GetOrganizationId() == "" {
		return fmt.Errorf("organization user perspective requires organization_id")
	}
	return validateGeneration(value.GetOrganizationId(), value.GetMembershipVersion(), currentMembershipVersion)
}

func ValidateRequestAccessScope(value *requestv1.RequestAccessScope, currentMembershipVersion int64) error {
	if value == nil {
		return fmt.Errorf("request access scope is required")
	}
	switch value.GetApp() {
	case requestv1.RequestAccessApp_REQUEST_ACCESS_APP_CLIENT:
		if value.GetPerspective() != requestv1.RequestAccessPerspective_REQUEST_ACCESS_PERSPECTIVE_BUYER {
			return fmt.Errorf("CLIENT request scope must use BUYER perspective")
		}
		if value.GetOrganizationId() != "" {
			return fmt.Errorf("CLIENT buyer request scope cannot carry organization_id")
		}
	case requestv1.RequestAccessApp_REQUEST_ACCESS_APP_PARTNER:
		if value.GetPerspective() != requestv1.RequestAccessPerspective_REQUEST_ACCESS_PERSPECTIVE_BUYER_ORG &&
			value.GetPerspective() != requestv1.RequestAccessPerspective_REQUEST_ACCESS_PERSPECTIVE_SUPPLIER_ORG {
			return fmt.Errorf("PARTNER request scope must use BUYER_ORG or SUPPLIER_ORG perspective")
		}
		if value.GetOrganizationId() == "" {
			return fmt.Errorf("PARTNER organization request scope requires organization_id")
		}
	case requestv1.RequestAccessApp_REQUEST_ACCESS_APP_ADMIN:
		if value.GetPerspective() != requestv1.RequestAccessPerspective_REQUEST_ACCESS_PERSPECTIVE_SUPPORT {
			return fmt.Errorf("ADMIN request scope must use SUPPORT perspective")
		}
	default:
		return fmt.Errorf("request scope app %s is invalid", value.GetApp())
	}
	return validateGeneration(value.GetOrganizationId(), value.GetMembershipVersion(), currentMembershipVersion)
}

func ValidateBidAccessScope(value *bidv1.BidAccessScope, currentMembershipVersion int64) error {
	if value == nil {
		return fmt.Errorf("bid access scope is required")
	}
	switch value.GetApp() {
	case bidv1.BidAccessApp_BID_ACCESS_APP_CLIENT:
		if value.GetPerspective() != bidv1.BidAccessPerspective_BID_ACCESS_PERSPECTIVE_BUYER {
			return fmt.Errorf("CLIENT bid scope must use BUYER perspective")
		}
		if value.GetOrganizationId() != "" {
			return fmt.Errorf("CLIENT buyer bid scope cannot carry organization_id")
		}
	case bidv1.BidAccessApp_BID_ACCESS_APP_PARTNER:
		if value.GetPerspective() != bidv1.BidAccessPerspective_BID_ACCESS_PERSPECTIVE_BUYER_ORG &&
			value.GetPerspective() != bidv1.BidAccessPerspective_BID_ACCESS_PERSPECTIVE_SUPPLIER_ORG {
			return fmt.Errorf("PARTNER bid scope must use BUYER_ORG or SUPPLIER_ORG perspective")
		}
		if value.GetOrganizationId() == "" {
			return fmt.Errorf("PARTNER organization bid scope requires organization_id")
		}
	case bidv1.BidAccessApp_BID_ACCESS_APP_ADMIN:
		if value.GetPerspective() != bidv1.BidAccessPerspective_BID_ACCESS_PERSPECTIVE_SUPPORT {
			return fmt.Errorf("ADMIN bid scope must use SUPPORT perspective")
		}
	default:
		return fmt.Errorf("bid scope app %s is invalid", value.GetApp())
	}
	return validateGeneration(value.GetOrganizationId(), value.GetMembershipVersion(), currentMembershipVersion)
}

func userAppScopeIdentity(value *userv1.UserAppScope) string {
	return fmt.Sprintf("%d/%d/%s/%d", value.GetApp(), value.GetPerspective(), value.GetOrganizationId(), value.GetMembershipVersion())
}

func requestAccessScopeIdentity(value *requestv1.RequestAccessScope) string {
	return fmt.Sprintf("%d/%d/%s/%d", value.GetApp(), value.GetPerspective(), value.GetOrganizationId(), value.GetMembershipVersion())
}

func bidAccessScopeIdentity(value *bidv1.BidAccessScope) string {
	return fmt.Sprintf("%d/%d/%s/%d", value.GetApp(), value.GetPerspective(), value.GetOrganizationId(), value.GetMembershipVersion())
}

func validateGeneration(organizationID string, membershipVersion, currentMembershipVersion int64) error {
	if organizationID == "" {
		if membershipVersion != 0 {
			return fmt.Errorf("membership_version must be 0 without organization_id")
		}
		return nil
	}
	if membershipVersion <= 0 {
		return fmt.Errorf("membership_version must be positive for organization_id")
	}
	if currentMembershipVersion <= 0 {
		return fmt.Errorf("current membership_version is required for organization_id")
	}
	if membershipVersion != currentMembershipVersion {
		return fmt.Errorf("stale membership_version %d; current is %d", membershipVersion, currentMembershipVersion)
	}
	return nil
}
