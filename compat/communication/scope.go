package communication

import (
	"fmt"
	"sort"
	"strings"

	chatv1 "github.com/4ubak/cg-proto/gen/go/communication/chat"
	notificationv1 "github.com/4ubak/cg-proto/gen/go/communication/notification"
)

// NotificationAudience is the normalized result for one push event. Each item
// in Scopes is a separate delivery; consumers must never compute a cross product.
type NotificationAudience struct {
	RecipientUserID int64
	Scopes          []*notificationv1.NotificationScope
	LegacyBroadcast bool
}

// NormalizePushAudience validates new bound tuples against every legacy form.
// New tuples are authoritative when present, deprecated typed fields are the
// next fallback, and exact legacy strings are the final migration fallback.
func NormalizePushAudience(event *notificationv1.PushEventPayload) (NotificationAudience, error) {
	if event == nil {
		return NotificationAudience{}, errNilLegacyEnvelope
	}
	if err := validateNotificationCategory(event.GetCategory(), event.GetTypedCategory()); err != nil {
		return NotificationAudience{}, err
	}

	legacyApps, err := normalizeLegacyNotificationApps(event.GetTargetApps())
	if err != nil {
		return NotificationAudience{}, err
	}
	boundScopes, err := validateNotificationScopes(event.GetRecipientScopes(), true, false)
	if err != nil {
		return NotificationAudience{}, err
	}
	if len(boundScopes) > 0 && event.GetUserId() <= 0 {
		return NotificationAudience{}, fmt.Errorf("canonical notification recipient user_id must be positive")
	}
	parallelScopes, parallelPresent, err := notificationParallelScopes(event)
	if err != nil {
		return NotificationAudience{}, err
	}

	switch {
	case len(boundScopes) > 0:
		if parallelPresent && !equalNotificationScopeIdentities(boundScopes, parallelScopes) {
			return NotificationAudience{}, fmt.Errorf("recipient_scopes conflict with deprecated typed routing")
		}
		if len(legacyApps) > 0 && !equalNotificationAppSets(boundScopes, legacyApps) {
			return NotificationAudience{}, fmt.Errorf("recipient_scopes conflict with target_apps")
		}
		return NotificationAudience{RecipientUserID: event.GetUserId(), Scopes: boundScopes}, nil
	case parallelPresent:
		if notificationScopesContainOrganization(parallelScopes) {
			return NotificationAudience{}, fmt.Errorf("deprecated typed organization routing has no membership_version")
		}
		if len(legacyApps) > 0 && !equalNotificationAppSets(parallelScopes, legacyApps) {
			return NotificationAudience{}, fmt.Errorf("deprecated typed routing conflicts with target_apps")
		}
		return NotificationAudience{Scopes: parallelScopes}, nil
	case len(legacyApps) > 0:
		scopes := make([]*notificationv1.NotificationScope, 0, len(legacyApps))
		for _, app := range legacyApps {
			scopes = append(scopes, &notificationv1.NotificationScope{App: app})
		}
		return NotificationAudience{Scopes: scopes}, nil
	default:
		return NotificationAudience{LegacyBroadcast: true}, nil
	}
}

func notificationParallelScopes(event *notificationv1.PushEventPayload) ([]*notificationv1.NotificationScope, bool, error) {
	present := len(event.GetTypedTargetApps()) > 0 ||
		event.GetRecipientPerspective() != notificationv1.NotificationPerspective_NOTIFICATION_PERSPECTIVE_UNSPECIFIED ||
		event.GetRecipientOrganizationId() != ""
	if !present {
		return nil, false, nil
	}
	if len(event.GetTypedTargetApps()) == 0 {
		return nil, true, fmt.Errorf("deprecated typed routing is partial: typed_target_apps is empty")
	}
	scopes := make([]*notificationv1.NotificationScope, 0, len(event.GetTypedTargetApps()))
	for _, app := range event.GetTypedTargetApps() {
		scopes = append(scopes, &notificationv1.NotificationScope{
			App:            app,
			Perspective:    event.GetRecipientPerspective(),
			OrganizationId: event.GetRecipientOrganizationId(),
		})
	}
	validated, err := validateNotificationScopes(scopes, false, true)
	return validated, true, err
}

func validateNotificationScopes(scopes []*notificationv1.NotificationScope, requirePerspective, allowMissingMembershipVersion bool) ([]*notificationv1.NotificationScope, error) {
	seen := make(map[string]int64, len(scopes))
	result := make([]*notificationv1.NotificationScope, 0, len(scopes))
	for _, scope := range scopes {
		if scope == nil {
			return nil, fmt.Errorf("notification scope is nil")
		}
		if scope.GetApp() != notificationv1.NotificationApp_NOTIFICATION_APP_CLIENT &&
			scope.GetApp() != notificationv1.NotificationApp_NOTIFICATION_APP_PRO {
			return nil, fmt.Errorf("notification scope app %s is invalid", scope.GetApp())
		}
		if requirePerspective && scope.GetPerspective() == notificationv1.NotificationPerspective_NOTIFICATION_PERSPECTIVE_UNSPECIFIED {
			return nil, fmt.Errorf("notification scope perspective is required")
		}
		switch scope.GetApp() {
		case notificationv1.NotificationApp_NOTIFICATION_APP_CLIENT:
			if scope.GetPerspective() != notificationv1.NotificationPerspective_NOTIFICATION_PERSPECTIVE_UNSPECIFIED &&
				scope.GetPerspective() != notificationv1.NotificationPerspective_NOTIFICATION_PERSPECTIVE_BUYER {
				return nil, fmt.Errorf("CLIENT notification scope must use BUYER perspective")
			}
			if scope.GetOrganizationId() != "" {
				return nil, fmt.Errorf("CLIENT buyer notification scope cannot carry organization_id")
			}
		case notificationv1.NotificationApp_NOTIFICATION_APP_PRO:
			if scope.GetPerspective() != notificationv1.NotificationPerspective_NOTIFICATION_PERSPECTIVE_UNSPECIFIED &&
				scope.GetPerspective() != notificationv1.NotificationPerspective_NOTIFICATION_PERSPECTIVE_BUYER_ORG &&
				scope.GetPerspective() != notificationv1.NotificationPerspective_NOTIFICATION_PERSPECTIVE_SELLER_ORG &&
				scope.GetPerspective() != notificationv1.NotificationPerspective_NOTIFICATION_PERSPECTIVE_SELLER_USER {
				return nil, fmt.Errorf("PRO notification scope must use BUYER_ORG or a seller perspective")
			}
		}
		if (scope.GetPerspective() == notificationv1.NotificationPerspective_NOTIFICATION_PERSPECTIVE_BUYER_ORG ||
			scope.GetPerspective() == notificationv1.NotificationPerspective_NOTIFICATION_PERSPECTIVE_SELLER_ORG) && scope.GetOrganizationId() == "" {
			return nil, fmt.Errorf("organization notification perspective requires organization_id")
		}
		if err := validateMembershipShape(scope.GetOrganizationId(), scope.GetMembershipVersion(), allowMissingMembershipVersion); err != nil {
			return nil, fmt.Errorf("notification scope: %w", err)
		}
		identity := notificationScopeIdentityKey(scope)
		if version, ok := seen[identity]; ok {
			if version != scope.GetMembershipVersion() {
				return nil, fmt.Errorf("notification scope %s has conflicting membership versions %d and %d", identity, version, scope.GetMembershipVersion())
			}
			return nil, fmt.Errorf("duplicate notification scope %s", identity)
		}
		seen[identity] = scope.GetMembershipVersion()
		result = append(result, scope)
	}
	return result, nil
}

func normalizeLegacyNotificationApps(values []string) ([]notificationv1.NotificationApp, error) {
	seen := make(map[notificationv1.NotificationApp]struct{}, len(values))
	apps := make([]notificationv1.NotificationApp, 0, len(values))
	for _, value := range values {
		var app notificationv1.NotificationApp
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "client":
			app = notificationv1.NotificationApp_NOTIFICATION_APP_CLIENT
		case "partner":
			app = notificationv1.NotificationApp_NOTIFICATION_APP_PRO
		default:
			return nil, fmt.Errorf("unsupported legacy notification app %q", value)
		}
		if _, ok := seen[app]; !ok {
			seen[app] = struct{}{}
			apps = append(apps, app)
		}
	}
	return apps, nil
}

func validateNotificationCategory(legacy string, typed notificationv1.NotificationCategory) error {
	if legacy == "" || typed == notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_UNSPECIFIED {
		return nil
	}
	want := map[string]notificationv1.NotificationCategory{
		"chat":    notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_CHAT,
		"order":   notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_ORDER,
		"promo":   notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_PROMO,
		"system":  notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_SYSTEM,
		"job":     notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_JOB,
		"payment": notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_PAYMENT,
		"cart":    notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_CART,
	}[strings.ToLower(strings.TrimSpace(legacy))]
	if want == notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_UNSPECIFIED || want != typed {
		return fmt.Errorf("typed_category %s conflicts with legacy category %q", typed, legacy)
	}
	return nil
}

func equalNotificationScopeIdentities(left, right []*notificationv1.NotificationScope) bool {
	return equalSortedStrings(notificationScopeIdentityKeys(left), notificationScopeIdentityKeys(right))
}

func equalNotificationAppSets(scopes []*notificationv1.NotificationScope, apps []notificationv1.NotificationApp) bool {
	leftSet := make(map[notificationv1.NotificationApp]struct{}, len(scopes))
	for _, scope := range scopes {
		leftSet[scope.GetApp()] = struct{}{}
	}
	rightSet := make(map[notificationv1.NotificationApp]struct{}, len(apps))
	for _, app := range apps {
		rightSet[app] = struct{}{}
	}
	if len(leftSet) != len(rightSet) {
		return false
	}
	for app := range leftSet {
		if _, ok := rightSet[app]; !ok {
			return false
		}
	}
	return true
}

func notificationScopeIdentityKeys(scopes []*notificationv1.NotificationScope) []string {
	keys := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		keys = append(keys, notificationScopeIdentityKey(scope))
	}
	return keys
}

func notificationScopeIdentityKey(scope *notificationv1.NotificationScope) string {
	return fmt.Sprintf("%d/%d/%s", scope.GetApp(), scope.GetPerspective(), scope.GetOrganizationId())
}

func notificationScopesContainOrganization(scopes []*notificationv1.NotificationScope) bool {
	for _, scope := range scopes {
		if scope.GetOrganizationId() != "" {
			return true
		}
	}
	return false
}

// ValidateNotificationMembershipVersion compares a normalized event scope with
// the current source-of-truth membership generation.
func ValidateNotificationMembershipVersion(scope *notificationv1.NotificationScope, currentVersion int64) error {
	if scope == nil {
		return fmt.Errorf("notification scope is nil")
	}
	if err := validateMembershipShape(scope.GetOrganizationId(), scope.GetMembershipVersion(), false); err != nil {
		return err
	}
	if scope.GetMembershipVersion() != currentVersion {
		return fmt.Errorf("stale notification membership_version %d; current is %d", scope.GetMembershipVersion(), currentVersion)
	}
	return nil
}

// ChatAudience is the normalized result for chat.message.sent routing.
type ChatAudience struct {
	Recipients      []*chatv1.ChatRecipient
	Scopes          []*chatv1.ChatScope
	LegacyBroadcast bool
}

// NormalizeChatAudience validates the single bound typed scope against the
// deprecated parallel fields and exact legacy target_apps/recipient_org_id.
func NormalizeChatAudience(event *chatv1.ChatRealtimeEventPayload) (ChatAudience, error) {
	if event == nil {
		return ChatAudience{}, errNilLegacyEnvelope
	}
	legacyApps, err := normalizeLegacyChatApps(event.GetTargetApps())
	if err != nil {
		return ChatAudience{}, err
	}
	legacyUsers, err := normalizeLegacyChatUsers(event.GetRecipientId(), event.GetRecipientUserIds())
	if err != nil {
		return ChatAudience{}, err
	}
	boundRecipients, err := validateChatRecipients(event.GetRecipients())
	if err != nil {
		return ChatAudience{}, err
	}
	bound := event.GetRecipientScope()
	if bound != nil {
		if err := validateChatScope(bound, true, false); err != nil {
			return ChatAudience{}, err
		}
	}
	parallel, parallelPresent, err := chatParallelScope(event)
	if err != nil {
		return ChatAudience{}, err
	}

	if len(boundRecipients) > 0 {
		if len(legacyUsers) > 0 && !equalInt64Sets(chatRecipientUserIDs(boundRecipients), legacyUsers) {
			return ChatAudience{}, fmt.Errorf("recipients conflict with legacy recipient identifiers")
		}
		if bound != nil {
			if bound.GetOrganizationId() != "" && len(legacyUsers) > 1 {
				return ChatAudience{}, fmt.Errorf("one organization recipient_scope cannot address multiple recipient_user_ids")
			}
			for _, recipient := range boundRecipients {
				if chatScopeCanonicalKey(recipient.GetScope()) != chatScopeCanonicalKey(bound) {
					return ChatAudience{}, fmt.Errorf("recipients conflict with recipient_scope")
				}
			}
		}
		if parallelPresent {
			for _, recipient := range boundRecipients {
				if chatScopeIdentityKey(recipient.GetScope()) != chatScopeIdentityKey(parallel) {
					return ChatAudience{}, fmt.Errorf("recipients conflict with deprecated typed routing")
				}
			}
		}
		if len(legacyApps) > 0 && !equalChatAppSets(chatRecipientScopes(boundRecipients), legacyApps) {
			return ChatAudience{}, fmt.Errorf("recipients conflict with target_apps")
		}
		if event.GetRecipientOrgId() != "" {
			for _, recipient := range boundRecipients {
				if recipient.GetScope().GetOrganizationId() != event.GetRecipientOrgId() {
					return ChatAudience{}, fmt.Errorf("recipients conflict with recipient_org_id")
				}
			}
		}
		return ChatAudience{Recipients: boundRecipients, Scopes: chatRecipientScopes(boundRecipients)}, nil
	}

	if bound != nil {
		if bound.GetOrganizationId() != "" && len(legacyUsers) > 1 {
			return ChatAudience{}, fmt.Errorf("one organization recipient_scope cannot address multiple recipient_user_ids")
		}
		if parallelPresent && chatScopeIdentityKey(bound) != chatScopeIdentityKey(parallel) {
			return ChatAudience{}, fmt.Errorf("recipient_scope conflicts with deprecated typed routing")
		}
		if len(legacyApps) > 1 || len(legacyApps) == 1 && legacyApps[0] != bound.GetApp() {
			return ChatAudience{}, fmt.Errorf("recipient_scope conflicts with target_apps")
		}
		if event.GetRecipientOrgId() != "" && event.GetRecipientOrgId() != bound.GetOrganizationId() {
			return ChatAudience{}, fmt.Errorf("recipient_scope conflicts with recipient_org_id")
		}
		return legacyChatAudience(bound, legacyUsers), nil
	}
	if parallelPresent {
		if parallel.GetOrganizationId() != "" {
			return ChatAudience{}, fmt.Errorf("deprecated typed chat organization routing has no membership_version")
		}
		if len(legacyApps) > 1 || len(legacyApps) == 1 && legacyApps[0] != parallel.GetApp() {
			return ChatAudience{}, fmt.Errorf("deprecated typed routing conflicts with target_apps")
		}
		if event.GetRecipientOrgId() != "" && event.GetRecipientOrgId() != parallel.GetOrganizationId() {
			return ChatAudience{}, fmt.Errorf("deprecated typed routing conflicts with recipient_org_id")
		}
		return legacyChatAudience(parallel, legacyUsers), nil
	}
	if len(legacyApps) == 0 {
		if event.GetRecipientOrgId() != "" {
			return ChatAudience{}, fmt.Errorf("legacy chat organization routing has no membership_version")
		}
		return ChatAudience{LegacyBroadcast: true}, nil
	}
	if event.GetRecipientOrgId() != "" {
		return ChatAudience{}, fmt.Errorf("legacy chat organization routing has no membership_version")
	}
	scopes := make([]*chatv1.ChatScope, 0, len(legacyApps))
	for _, app := range legacyApps {
		scopes = append(scopes, &chatv1.ChatScope{App: app, OrganizationId: event.GetRecipientOrgId()})
	}
	if len(scopes) == 1 {
		return legacyChatAudience(scopes[0], legacyUsers), nil
	}
	return ChatAudience{Scopes: scopes}, nil
}

func validateChatRecipients(recipients []*chatv1.ChatRecipient) ([]*chatv1.ChatRecipient, error) {
	seenUsers := make(map[int64]string, len(recipients))
	result := make([]*chatv1.ChatRecipient, 0, len(recipients))
	for _, recipient := range recipients {
		if recipient == nil {
			return nil, fmt.Errorf("chat recipient is nil")
		}
		if recipient.GetUserId() <= 0 {
			return nil, fmt.Errorf("chat recipient user_id must be positive")
		}
		if err := validateChatScope(recipient.GetScope(), true, false); err != nil {
			return nil, fmt.Errorf("chat recipient %d: %w", recipient.GetUserId(), err)
		}
		canonical := chatScopeCanonicalKey(recipient.GetScope())
		if previous, duplicate := seenUsers[recipient.GetUserId()]; duplicate {
			if previous != canonical {
				return nil, fmt.Errorf("chat recipient user_id %d has conflicting scopes", recipient.GetUserId())
			}
			return nil, fmt.Errorf("duplicate chat recipient user_id %d", recipient.GetUserId())
		}
		seenUsers[recipient.GetUserId()] = canonical
		result = append(result, recipient)
	}
	return result, nil
}

func normalizeLegacyChatUsers(recipientID int64, recipientUserIDs []int64) ([]int64, error) {
	seen := make(map[int64]struct{}, len(recipientUserIDs)+1)
	result := make([]int64, 0, len(recipientUserIDs)+1)
	for _, userID := range append([]int64{recipientID}, recipientUserIDs...) {
		if userID == 0 {
			continue
		}
		if userID < 0 {
			return nil, fmt.Errorf("legacy recipient user_id must be positive")
		}
		if _, duplicate := seen[userID]; duplicate {
			continue
		}
		seen[userID] = struct{}{}
		result = append(result, userID)
	}
	return result, nil
}

func legacyChatAudience(scope *chatv1.ChatScope, userIDs []int64) ChatAudience {
	audience := ChatAudience{Scopes: []*chatv1.ChatScope{scope}}
	for _, userID := range userIDs {
		audience.Recipients = append(audience.Recipients, &chatv1.ChatRecipient{UserId: userID, Scope: scope})
	}
	return audience
}

func chatRecipientScopes(recipients []*chatv1.ChatRecipient) []*chatv1.ChatScope {
	scopes := make([]*chatv1.ChatScope, 0, len(recipients))
	for _, recipient := range recipients {
		scopes = append(scopes, recipient.GetScope())
	}
	return scopes
}

func chatRecipientUserIDs(recipients []*chatv1.ChatRecipient) []int64 {
	result := make([]int64, 0, len(recipients))
	for _, recipient := range recipients {
		result = append(result, recipient.GetUserId())
	}
	return result
}

func equalChatAppSets(scopes []*chatv1.ChatScope, apps []chatv1.ChatApp) bool {
	left := make(map[chatv1.ChatApp]struct{}, len(scopes))
	for _, scope := range scopes {
		left[scope.GetApp()] = struct{}{}
	}
	right := make(map[chatv1.ChatApp]struct{}, len(apps))
	for _, app := range apps {
		right[app] = struct{}{}
	}
	if len(left) != len(right) {
		return false
	}
	for app := range left {
		if _, ok := right[app]; !ok {
			return false
		}
	}
	return true
}

func equalInt64Sets(left, right []int64) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy := append([]int64(nil), left...)
	rightCopy := append([]int64(nil), right...)
	sort.Slice(leftCopy, func(i, j int) bool { return leftCopy[i] < leftCopy[j] })
	sort.Slice(rightCopy, func(i, j int) bool { return rightCopy[i] < rightCopy[j] })
	for i := range leftCopy {
		if leftCopy[i] != rightCopy[i] {
			return false
		}
	}
	return true
}

func chatParallelScope(event *chatv1.ChatRealtimeEventPayload) (*chatv1.ChatScope, bool, error) {
	present := event.GetRecipientApp() != chatv1.ChatApp_CHAT_APP_UNSPECIFIED ||
		event.GetRecipientPerspective() != chatv1.ChatPerspective_CHAT_PERSPECTIVE_UNSPECIFIED ||
		event.GetRecipientOrganizationId() != ""
	if !present {
		return nil, false, nil
	}
	if event.GetRecipientApp() == chatv1.ChatApp_CHAT_APP_UNSPECIFIED {
		return nil, true, fmt.Errorf("deprecated typed chat routing is partial: recipient_app is unspecified")
	}
	scope := &chatv1.ChatScope{
		App:            event.GetRecipientApp(),
		Perspective:    event.GetRecipientPerspective(),
		OrganizationId: event.GetRecipientOrganizationId(),
	}
	if err := validateChatScope(scope, false, true); err != nil {
		return nil, true, err
	}
	return scope, true, nil
}

func normalizeLegacyChatApps(values []string) ([]chatv1.ChatApp, error) {
	seen := make(map[chatv1.ChatApp]struct{}, len(values))
	apps := make([]chatv1.ChatApp, 0, len(values))
	for _, value := range values {
		var app chatv1.ChatApp
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "client":
			app = chatv1.ChatApp_CHAT_APP_CLIENT
		case "partner":
			app = chatv1.ChatApp_CHAT_APP_PRO
		default:
			return nil, fmt.Errorf("unsupported legacy chat app %q", value)
		}
		if _, ok := seen[app]; !ok {
			seen[app] = struct{}{}
			apps = append(apps, app)
		}
	}
	return apps, nil
}

func validateChatScope(scope *chatv1.ChatScope, requirePerspective, allowMissingMembershipVersion bool) error {
	if scope == nil {
		return fmt.Errorf("chat scope is nil")
	}
	if requirePerspective && scope.GetPerspective() == chatv1.ChatPerspective_CHAT_PERSPECTIVE_UNSPECIFIED {
		return fmt.Errorf("chat scope perspective is required")
	}
	switch scope.GetApp() {
	case chatv1.ChatApp_CHAT_APP_CLIENT:
		if scope.GetPerspective() != chatv1.ChatPerspective_CHAT_PERSPECTIVE_UNSPECIFIED &&
			scope.GetPerspective() != chatv1.ChatPerspective_CHAT_PERSPECTIVE_BUYER {
			return fmt.Errorf("CLIENT chat scope must use BUYER perspective")
		}
	case chatv1.ChatApp_CHAT_APP_PRO:
		if scope.GetPerspective() != chatv1.ChatPerspective_CHAT_PERSPECTIVE_UNSPECIFIED &&
			scope.GetPerspective() != chatv1.ChatPerspective_CHAT_PERSPECTIVE_BUYER_ORG &&
			scope.GetPerspective() != chatv1.ChatPerspective_CHAT_PERSPECTIVE_SELLER_ORG &&
			scope.GetPerspective() != chatv1.ChatPerspective_CHAT_PERSPECTIVE_SELLER_USER {
			return fmt.Errorf("PRO chat scope must use BUYER_ORG or a seller perspective")
		}
	case chatv1.ChatApp_CHAT_APP_ADMIN:
		if scope.GetPerspective() != chatv1.ChatPerspective_CHAT_PERSPECTIVE_SUPPORT {
			return fmt.Errorf("ADMIN chat scope must use SUPPORT perspective")
		}
	default:
		return fmt.Errorf("chat scope app %s is invalid", scope.GetApp())
	}
	if (scope.GetPerspective() == chatv1.ChatPerspective_CHAT_PERSPECTIVE_BUYER_ORG ||
		scope.GetPerspective() == chatv1.ChatPerspective_CHAT_PERSPECTIVE_SELLER_ORG) && scope.GetOrganizationId() == "" {
		return fmt.Errorf("organization chat perspective requires organization_id")
	}
	if err := validateMembershipShape(scope.GetOrganizationId(), scope.GetMembershipVersion(), allowMissingMembershipVersion); err != nil {
		return fmt.Errorf("chat scope: %w", err)
	}
	return nil
}

func chatScopeIdentityKey(scope *chatv1.ChatScope) string {
	return fmt.Sprintf("%d/%d/%s", scope.GetApp(), scope.GetPerspective(), scope.GetOrganizationId())
}

func chatScopeCanonicalKey(scope *chatv1.ChatScope) string {
	return fmt.Sprintf("%s/%d", chatScopeIdentityKey(scope), scope.GetMembershipVersion())
}

// ValidateChatMembershipVersion compares a normalized event scope with the
// current source-of-truth membership generation.
func ValidateChatMembershipVersion(scope *chatv1.ChatScope, currentVersion int64) error {
	if scope == nil {
		return fmt.Errorf("chat scope is nil")
	}
	if err := validateMembershipShape(scope.GetOrganizationId(), scope.GetMembershipVersion(), false); err != nil {
		return err
	}
	if scope.GetMembershipVersion() != currentVersion {
		return fmt.Errorf("stale chat membership_version %d; current is %d", scope.GetMembershipVersion(), currentVersion)
	}
	return nil
}

func validateMembershipShape(organizationID string, membershipVersion int64, allowMissingOrganizationVersion bool) error {
	switch {
	case organizationID == "" && membershipVersion != 0:
		return fmt.Errorf("membership_version must be 0 without organization_id")
	case organizationID != "" && membershipVersion < 0:
		return fmt.Errorf("membership_version must be positive for organization_id")
	case organizationID != "" && membershipVersion == 0 && !allowMissingOrganizationVersion:
		return fmt.Errorf("membership_version must be positive for organization_id")
	default:
		return nil
	}
}

func equalSortedStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy := append([]string(nil), left...)
	rightCopy := append([]string(nil), right...)
	sort.Strings(leftCopy)
	sort.Strings(rightCopy)
	for i := range leftCopy {
		if leftCopy[i] != rightCopy[i] {
			return false
		}
	}
	return true
}
