package communication

import (
	"testing"

	chatv1 "github.com/4ubak/cg-proto/gen/go/communication/chat"
	notificationv1 "github.com/4ubak/cg-proto/gen/go/communication/notification"
)

func TestNormalizePushAudience(t *testing.T) {
	client := &notificationv1.NotificationScope{
		App:         notificationv1.NotificationApp_NOTIFICATION_APP_CLIENT,
		Perspective: notificationv1.NotificationPerspective_NOTIFICATION_PERSPECTIVE_BUYER,
	}
	partner := &notificationv1.NotificationScope{
		App:               notificationv1.NotificationApp_NOTIFICATION_APP_PRO,
		Perspective:       notificationv1.NotificationPerspective_NOTIFICATION_PERSPECTIVE_SELLER_ORG,
		OrganizationId:    "org-1",
		MembershipVersion: 7,
	}
	partnerBuyer := &notificationv1.NotificationScope{
		App:               notificationv1.NotificationApp_NOTIFICATION_APP_PRO,
		Perspective:       notificationv1.NotificationPerspective_NOTIFICATION_PERSPECTIVE_BUYER_ORG,
		OrganizationId:    "org-1",
		MembershipVersion: 7,
	}

	t.Run("PRO buyer organization is distinct from supplier organization", func(t *testing.T) {
		result, err := NormalizePushAudience(&notificationv1.PushEventPayload{UserId: 27, RecipientScopes: []*notificationv1.NotificationScope{partnerBuyer, partner}})
		if err != nil {
			t.Fatalf("normalize: %v", err)
		}
		if len(result.Scopes) != 2 || notificationScopeIdentityKey(result.Scopes[0]) == notificationScopeIdentityKey(result.Scopes[1]) {
			t.Fatalf("buyer/supplier scopes were mixed: %+v", result.Scopes)
		}
	})

	t.Run("canonical notification scope requires its bound user", func(t *testing.T) {
		if _, err := NormalizePushAudience(&notificationv1.PushEventPayload{RecipientScopes: []*notificationv1.NotificationScope{partner}}); err == nil {
			t.Fatal("canonical notification tuple accepted without user_id")
		}
	})

	t.Run("matching multiple bound scopes stay separate", func(t *testing.T) {
		result, err := NormalizePushAudience(&notificationv1.PushEventPayload{
			UserId:          27,
			TargetApps:      []string{"client", "partner"},
			RecipientScopes: []*notificationv1.NotificationScope{client, partner},
		})
		if err != nil {
			t.Fatalf("normalize: %v", err)
		}
		if result.LegacyBroadcast || len(result.Scopes) != 2 || result.Scopes[0] != client || result.Scopes[1] != partner {
			t.Fatalf("bound tuples were flattened or reordered: %+v", result)
		}
	})

	t.Run("typed legacy conflict is rejected", func(t *testing.T) {
		_, err := NormalizePushAudience(&notificationv1.PushEventPayload{
			UserId:          27,
			TargetApps:      []string{"client"},
			RecipientScopes: []*notificationv1.NotificationScope{partner},
		})
		if err == nil {
			t.Fatal("expected app conflict")
		}
	})

	t.Run("deprecated parallel conflict is rejected", func(t *testing.T) {
		_, err := NormalizePushAudience(&notificationv1.PushEventPayload{
			UserId:                  27,
			TypedTargetApps:         []notificationv1.NotificationApp{notificationv1.NotificationApp_NOTIFICATION_APP_PRO},
			RecipientPerspective:    notificationv1.NotificationPerspective_NOTIFICATION_PERSPECTIVE_SELLER_ORG,
			RecipientOrganizationId: "org-2",
			RecipientScopes:         []*notificationv1.NotificationScope{partner},
		})
		if err == nil {
			t.Fatal("expected deprecated/new typed conflict")
		}
	})

	t.Run("same scope with conflicting membership versions is rejected", func(t *testing.T) {
		stale := &notificationv1.NotificationScope{
			App:               partner.GetApp(),
			Perspective:       partner.GetPerspective(),
			OrganizationId:    partner.GetOrganizationId(),
			MembershipVersion: partner.GetMembershipVersion() - 1,
		}
		_, err := NormalizePushAudience(&notificationv1.PushEventPayload{
			UserId:          27,
			RecipientScopes: []*notificationv1.NotificationScope{stale, partner},
		})
		if err == nil {
			t.Fatal("expected membership version conflict")
		}
	})

	t.Run("organization routing without canonical version is rejected", func(t *testing.T) {
		_, err := NormalizePushAudience(&notificationv1.PushEventPayload{
			UserId:                  27,
			TypedTargetApps:         []notificationv1.NotificationApp{notificationv1.NotificationApp_NOTIFICATION_APP_PRO},
			RecipientPerspective:    notificationv1.NotificationPerspective_NOTIFICATION_PERSPECTIVE_SELLER_ORG,
			RecipientOrganizationId: "org-1",
		})
		if err == nil {
			t.Fatal("expected missing membership_version rejection")
		}
	})

	t.Run("legacy only migrates deterministically", func(t *testing.T) {
		result, err := NormalizePushAudience(&notificationv1.PushEventPayload{TargetApps: []string{"partner"}})
		if err != nil {
			t.Fatalf("normalize: %v", err)
		}
		if result.LegacyBroadcast || len(result.Scopes) != 1 || result.Scopes[0].GetApp() != notificationv1.NotificationApp_NOTIFICATION_APP_PRO {
			t.Fatalf("legacy normalization = %+v", result)
		}
	})

	t.Run("empty legacy keeps broadcast", func(t *testing.T) {
		result, err := NormalizePushAudience(&notificationv1.PushEventPayload{})
		if err != nil {
			t.Fatalf("normalize: %v", err)
		}
		if !result.LegacyBroadcast || len(result.Scopes) != 0 {
			t.Fatalf("empty legacy audience = %+v", result)
		}
	})

	t.Run("category conflict is rejected", func(t *testing.T) {
		_, err := NormalizePushAudience(&notificationv1.PushEventPayload{
			Category:      "promo",
			TypedCategory: notificationv1.NotificationCategory_NOTIFICATION_CATEGORY_SYSTEM,
		})
		if err == nil {
			t.Fatal("expected category conflict")
		}
	})

	t.Run("current membership generation is enforced", func(t *testing.T) {
		if err := ValidateNotificationMembershipVersion(partner, 7); err != nil {
			t.Fatalf("current membership rejected: %v", err)
		}
		if err := ValidateNotificationMembershipVersion(partner, 8); err == nil {
			t.Fatal("stale membership version accepted")
		}
	})
}

func TestNormalizeChatAudience(t *testing.T) {
	partner := &chatv1.ChatScope{
		App:               chatv1.ChatApp_CHAT_APP_PRO,
		Perspective:       chatv1.ChatPerspective_CHAT_PERSPECTIVE_SELLER_ORG,
		OrganizationId:    "org-1",
		MembershipVersion: 11,
	}
	partnerBuyer := &chatv1.ChatScope{
		App:               chatv1.ChatApp_CHAT_APP_PRO,
		Perspective:       chatv1.ChatPerspective_CHAT_PERSPECTIVE_BUYER_ORG,
		OrganizationId:    "org-buyer",
		MembershipVersion: 4,
	}

	t.Run("PRO buyer organization does not alias supplier", func(t *testing.T) {
		result, err := NormalizeChatAudience(&chatv1.ChatRealtimeEventPayload{Recipients: []*chatv1.ChatRecipient{{UserId: 30, Scope: partnerBuyer}}})
		if err != nil {
			t.Fatalf("normalize: %v", err)
		}
		if len(result.Recipients) != 1 || result.Recipients[0].GetScope().GetPerspective() != chatv1.ChatPerspective_CHAT_PERSPECTIVE_BUYER_ORG || chatScopeIdentityKey(partnerBuyer) == chatScopeIdentityKey(partner) {
			t.Fatalf("buyer organization scope was mixed: %+v", result)
		}
	})

	t.Run("organization fanout binds each user generation", func(t *testing.T) {
		first := &chatv1.ChatRecipient{UserId: 41, Scope: partner}
		second := &chatv1.ChatRecipient{UserId: 42, Scope: &chatv1.ChatScope{App: partner.GetApp(), Perspective: partner.GetPerspective(), OrganizationId: partner.GetOrganizationId(), MembershipVersion: 19}}
		result, err := NormalizeChatAudience(&chatv1.ChatRealtimeEventPayload{
			RecipientUserIds: []int64{41, 42},
			TargetApps:       []string{"partner"},
			RecipientOrgId:   "org-1",
			Recipients:       []*chatv1.ChatRecipient{first, second},
		})
		if err != nil {
			t.Fatalf("normalize: %v", err)
		}
		if len(result.Recipients) != 2 || result.Recipients[0].GetScope().GetMembershipVersion() != 11 || result.Recipients[1].GetScope().GetMembershipVersion() != 19 {
			t.Fatalf("recipient generations lost: %+v", result.Recipients)
		}
	})

	t.Run("bound recipients reject mixed app", func(t *testing.T) {
		_, err := NormalizeChatAudience(&chatv1.ChatRealtimeEventPayload{Recipients: []*chatv1.ChatRecipient{
			{UserId: 41, Scope: partner},
			{UserId: 42, Scope: &chatv1.ChatScope{App: chatv1.ChatApp_CHAT_APP_CLIENT, Perspective: chatv1.ChatPerspective_CHAT_PERSPECTIVE_BUYER}},
		}})
		if err == nil {
			t.Fatal("mixed recipient apps accepted")
		}
	})

	t.Run("bound recipients reject mixed organization", func(t *testing.T) {
		_, err := NormalizeChatAudience(&chatv1.ChatRealtimeEventPayload{Recipients: []*chatv1.ChatRecipient{
			{UserId: 41, Scope: partner},
			{UserId: 42, Scope: &chatv1.ChatScope{App: partner.GetApp(), Perspective: partner.GetPerspective(), OrganizationId: "org-2", MembershipVersion: 19}},
		}})
		if err == nil {
			t.Fatal("mixed recipient organizations accepted")
		}
	})

	t.Run("bound recipients reject mixed perspective", func(t *testing.T) {
		_, err := NormalizeChatAudience(&chatv1.ChatRealtimeEventPayload{Recipients: []*chatv1.ChatRecipient{
			{UserId: 41, Scope: partner},
			{UserId: 42, Scope: &chatv1.ChatScope{App: partner.GetApp(), Perspective: chatv1.ChatPerspective_CHAT_PERSPECTIVE_BUYER_ORG, OrganizationId: partner.GetOrganizationId(), MembershipVersion: 19}},
		}})
		if err == nil {
			t.Fatal("mixed recipient perspectives accepted")
		}
	})

	t.Run("multi recipient event rejects deprecated single scope", func(t *testing.T) {
		_, err := NormalizeChatAudience(&chatv1.ChatRealtimeEventPayload{
			RecipientScope: partner,
			Recipients: []*chatv1.ChatRecipient{
				{UserId: 41, Scope: partner},
				{UserId: 42, Scope: &chatv1.ChatScope{App: partner.GetApp(), Perspective: partner.GetPerspective(), OrganizationId: partner.GetOrganizationId(), MembershipVersion: 19}},
			},
		})
		if err == nil {
			t.Fatal("multi-recipient event accepted one deprecated recipient_scope")
		}
	})

	t.Run("single recipient scope must agree through membership generation", func(t *testing.T) {
		result, err := NormalizeChatAudience(&chatv1.ChatRealtimeEventPayload{
			RecipientId:    41,
			RecipientScope: partner,
			Recipients:     []*chatv1.ChatRecipient{{UserId: 41, Scope: partner}},
		})
		if err != nil || len(result.Recipients) != 1 {
			t.Fatalf("matching single recipient scope rejected: result=%+v err=%v", result, err)
		}
		stale := &chatv1.ChatScope{App: partner.GetApp(), Perspective: partner.GetPerspective(), OrganizationId: partner.GetOrganizationId(), MembershipVersion: partner.GetMembershipVersion() - 1}
		if _, err := NormalizeChatAudience(&chatv1.ChatRealtimeEventPayload{RecipientScope: stale, Recipients: []*chatv1.ChatRecipient{{UserId: 41, Scope: partner}}}); err == nil {
			t.Fatal("single recipient accepted conflicting recipient_scope generation")
		}
	})

	t.Run("non organization multi recipient follows the same single scope rule", func(t *testing.T) {
		client := &chatv1.ChatScope{App: chatv1.ChatApp_CHAT_APP_CLIENT, Perspective: chatv1.ChatPerspective_CHAT_PERSPECTIVE_BUYER}
		event := &chatv1.ChatRealtimeEventPayload{
			RecipientUserIds: []int64{51, 52},
			TargetApps:       []string{"client"},
			Recipients: []*chatv1.ChatRecipient{
				{UserId: 51, Scope: client},
				{UserId: 52, Scope: client},
			},
		}
		if _, err := NormalizeChatAudience(event); err != nil {
			t.Fatalf("homogeneous non-org fanout rejected: %v", err)
		}
		event.RecipientScope = client
		if _, err := NormalizeChatAudience(event); err == nil {
			t.Fatal("non-org multi-recipient event accepted one recipient_scope")
		}
	})

	t.Run("fired and rehired generation for one user conflicts", func(t *testing.T) {
		_, err := NormalizeChatAudience(&chatv1.ChatRealtimeEventPayload{Recipients: []*chatv1.ChatRecipient{
			{UserId: 41, Scope: partner},
			{UserId: 41, Scope: &chatv1.ChatScope{App: partner.GetApp(), Perspective: partner.GetPerspective(), OrganizationId: partner.GetOrganizationId(), MembershipVersion: 12}},
		}})
		if err == nil {
			t.Fatal("same user accepted under fired and rehired generations")
		}
	})

	t.Run("one legacy organization scope cannot fan out to many users", func(t *testing.T) {
		_, err := NormalizeChatAudience(&chatv1.ChatRealtimeEventPayload{RecipientUserIds: []int64{41, 42}, RecipientScope: partner})
		if err == nil {
			t.Fatal("single organization generation accepted for multiple users")
		}
	})

	t.Run("bound recipients must agree with legacy user identifiers", func(t *testing.T) {
		_, err := NormalizeChatAudience(&chatv1.ChatRealtimeEventPayload{
			RecipientUserIds: []int64{99},
			Recipients:       []*chatv1.ChatRecipient{{UserId: 41, Scope: partner}},
		})
		if err == nil {
			t.Fatal("conflicting canonical and legacy user identity accepted")
		}
	})

	t.Run("bound recipients must agree with legacy app", func(t *testing.T) {
		_, err := NormalizeChatAudience(&chatv1.ChatRealtimeEventPayload{
			TargetApps: []string{"client"},
			Recipients: []*chatv1.ChatRecipient{{UserId: 41, Scope: partner}},
		})
		if err == nil {
			t.Fatal("conflicting canonical and legacy app accepted")
		}
	})

	t.Run("bound recipients must agree with legacy organization", func(t *testing.T) {
		_, err := NormalizeChatAudience(&chatv1.ChatRealtimeEventPayload{
			RecipientOrgId: "org-other",
			Recipients:     []*chatv1.ChatRecipient{{UserId: 41, Scope: partner}},
		})
		if err == nil {
			t.Fatal("conflicting canonical and legacy organization accepted")
		}
	})

	t.Run("matching typed and legacy route", func(t *testing.T) {
		result, err := NormalizeChatAudience(&chatv1.ChatRealtimeEventPayload{
			TargetApps:     []string{"partner"},
			RecipientOrgId: "org-1",
			RecipientScope: partner,
		})
		if err != nil {
			t.Fatalf("normalize: %v", err)
		}
		if result.LegacyBroadcast || len(result.Scopes) != 1 || result.Scopes[0] != partner {
			t.Fatalf("chat scope = %+v", result)
		}
	})

	t.Run("cross app typed route is rejected", func(t *testing.T) {
		_, err := NormalizeChatAudience(&chatv1.ChatRealtimeEventPayload{
			TargetApps:     []string{"client", "partner"},
			RecipientOrgId: "org-1",
			RecipientScope: partner,
		})
		if err == nil {
			t.Fatal("expected multi-app legacy conflict")
		}
	})

	t.Run("organization conflict is rejected", func(t *testing.T) {
		_, err := NormalizeChatAudience(&chatv1.ChatRealtimeEventPayload{
			TargetApps:     []string{"partner"},
			RecipientOrgId: "org-other",
			RecipientScope: partner,
		})
		if err == nil {
			t.Fatal("expected organization conflict")
		}
	})

	t.Run("legacy non-org partner route migrates", func(t *testing.T) {
		result, err := NormalizeChatAudience(&chatv1.ChatRealtimeEventPayload{
			TargetApps: []string{"partner"},
		})
		if err != nil {
			t.Fatalf("normalize: %v", err)
		}
		if len(result.Scopes) != 1 || result.Scopes[0].GetApp() != chatv1.ChatApp_CHAT_APP_PRO ||
			result.Scopes[0].GetOrganizationId() != "" || result.Scopes[0].GetMembershipVersion() != 0 {
			t.Fatalf("legacy chat route = %+v", result)
		}
	})

	t.Run("legacy organization route without version is rejected", func(t *testing.T) {
		_, err := NormalizeChatAudience(&chatv1.ChatRealtimeEventPayload{
			TargetApps:     []string{"partner"},
			RecipientOrgId: "org-legacy",
		})
		if err == nil {
			t.Fatal("expected missing membership_version rejection")
		}
	})

	t.Run("empty legacy keeps broadcast", func(t *testing.T) {
		result, err := NormalizeChatAudience(&chatv1.ChatRealtimeEventPayload{})
		if err != nil {
			t.Fatalf("normalize: %v", err)
		}
		if !result.LegacyBroadcast || len(result.Scopes) != 0 {
			t.Fatalf("empty chat route = %+v", result)
		}
	})

	t.Run("support scope is representable only as admin support", func(t *testing.T) {
		result, err := NormalizeChatAudience(&chatv1.ChatRealtimeEventPayload{RecipientScope: &chatv1.ChatScope{
			App:         chatv1.ChatApp_CHAT_APP_ADMIN,
			Perspective: chatv1.ChatPerspective_CHAT_PERSPECTIVE_SUPPORT,
		}})
		if err != nil {
			t.Fatalf("normalize admin support: %v", err)
		}
		if len(result.Scopes) != 1 {
			t.Fatalf("support scope = %+v", result)
		}
	})

	t.Run("current membership generation is enforced", func(t *testing.T) {
		if err := ValidateChatMembershipVersion(partner, 11); err != nil {
			t.Fatalf("current membership rejected: %v", err)
		}
		if err := ValidateChatMembershipVersion(partner, 12); err == nil {
			t.Fatal("stale membership version accepted")
		}
		if err := ValidateChatMembershipVersion(partner, 0); err == nil {
			t.Fatal("removed member generation accepted")
		}
	})
}
