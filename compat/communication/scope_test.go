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

	t.Run("matching multiple bound scopes stay separate", func(t *testing.T) {
		result, err := NormalizePushAudience(&notificationv1.PushEventPayload{
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
			TargetApps:      []string{"client"},
			RecipientScopes: []*notificationv1.NotificationScope{partner},
		})
		if err == nil {
			t.Fatal("expected app conflict")
		}
	})

	t.Run("deprecated parallel conflict is rejected", func(t *testing.T) {
		_, err := NormalizePushAudience(&notificationv1.PushEventPayload{
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
			RecipientScopes: []*notificationv1.NotificationScope{stale, partner},
		})
		if err == nil {
			t.Fatal("expected membership version conflict")
		}
	})

	t.Run("organization routing without canonical version is rejected", func(t *testing.T) {
		_, err := NormalizePushAudience(&notificationv1.PushEventPayload{
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
	})
}
