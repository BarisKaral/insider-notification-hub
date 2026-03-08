package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/baris/notification-hub/config"
	"github.com/baris/notification-hub/internal/notification"
	"github.com/baris/notification-hub/internal/notification/template"
	"github.com/baris/notification-hub/pkg/postgres"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := postgres.NewPostgresDB(postgres.PostgresConfig{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		Name:     cfg.Database.Name,
		SSLMode:  cfg.Database.SSLMode,
	})
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	// Clean up existing seed data (soft-deleted or not) to make seeder idempotent.
	db.Unscoped().Where("recipient LIKE ?", "seed-%").Delete(&notification.Notification{})
	db.Unscoped().Where("name LIKE ?", "seed-%").Delete(&template.Template{})

	log.Println("Cleaned up existing seed data")

	// --- Templates ---

	smsTemplateID := uuid.New()
	emailTemplateID := uuid.New()
	pushTemplateID := uuid.New()

	templates := []template.Template{
		{
			ID:      smsTemplateID,
			Name:    "seed-sms-verification",
			Channel: "sms",
			Content: "Your verification code is {{.code}}. It expires in {{.expiry}} minutes.",
		},
		{
			ID:      emailTemplateID,
			Name:    "seed-email-welcome",
			Channel: "email",
			Content: "Hello {{.name}}, welcome to our platform! Your account has been created successfully.",
		},
		{
			ID:      pushTemplateID,
			Name:    "seed-push-order-update",
			Channel: "push",
			Content: "Your order #{{.orderId}} has been {{.status}}.",
		},
	}

	if err := db.Create(&templates).Error; err != nil {
		log.Fatalf("failed to create templates: %v", err)
	}
	log.Printf("Created %d templates", len(templates))

	// --- Individual Notifications (10 in various statuses) ---

	now := time.Now().UTC()
	past := now.Add(-2 * time.Hour)
	future := now.Add(1 * time.Hour)

	providerMsg := "provider-msg-12345"
	failureReason := "connection timeout: provider unreachable"
	idemKey1 := "seed-idem-001"
	idemKey2 := "seed-idem-002"

	smsVars := json.RawMessage(`{"code":"123456","expiry":"5"}`)
	emailVars := json.RawMessage(`{"name":"Alice"}`)
	pushVars := json.RawMessage(`{"orderId":"ORD-9001","status":"shipped"}`)

	notifications := []notification.Notification{
		{
			ID:        uuid.New(),
			Recipient: "seed-user-1@example.com",
			Channel:   notification.NotificationChannelEmail,
			Content:   "Welcome to our platform, Alice!",
			Priority:  notification.NotificationPriorityNormal,
			Status:    notification.NotificationStatusPending,
		},
		{
			ID:           uuid.New(),
			Recipient:    "seed-user-2@example.com",
			Channel:      notification.NotificationChannelEmail,
			Content:      "Your account has been verified.",
			Priority:     notification.NotificationPriorityHigh,
			Status:       notification.NotificationStatusSent,
			SentAt:       &past,
			ProviderMsgID: &providerMsg,
			TemplateID:   &emailTemplateID,
			TemplateVars: emailVars,
		},
		{
			ID:            uuid.New(),
			Recipient:     "seed-+905551234567",
			Channel:       notification.NotificationChannelSMS,
			Content:       "Your verification code is 123456.",
			Priority:      notification.NotificationPriorityHigh,
			Status:        notification.NotificationStatusFailed,
			FailedAt:      &past,
			FailureReason: &failureReason,
			RetryCount:    3,
			TemplateID:    &smsTemplateID,
			TemplateVars:  smsVars,
		},
		{
			ID:        uuid.New(),
			Recipient: "seed-device-token-abc",
			Channel:   notification.NotificationChannelPush,
			Content:   "Your order #ORD-9001 has been shipped.",
			Priority:  notification.NotificationPriorityNormal,
			Status:    notification.NotificationStatusQueued,
			TemplateID:   &pushTemplateID,
			TemplateVars: pushVars,
		},
		{
			ID:        uuid.New(),
			Recipient: "seed-+905559876543",
			Channel:   notification.NotificationChannelSMS,
			Content:   "Your appointment is confirmed for tomorrow.",
			Priority:  notification.NotificationPriorityLow,
			Status:    notification.NotificationStatusScheduled,
			ScheduledAt: &future,
		},
		{
			ID:        uuid.New(),
			Recipient: "seed-user-3@example.com",
			Channel:   notification.NotificationChannelEmail,
			Content:   "Password reset link: https://example.com/reset/abc",
			Priority:  notification.NotificationPriorityHigh,
			Status:    notification.NotificationStatusProcessing,
		},
		{
			ID:             uuid.New(),
			Recipient:      "seed-device-token-def",
			Channel:        notification.NotificationChannelPush,
			Content:        "Flash sale starts now! 50% off everything.",
			Priority:       notification.NotificationPriorityNormal,
			Status:         notification.NotificationStatusSent,
			SentAt:         &past,
			IdempotencyKey: &idemKey1,
		},
		{
			ID:            uuid.New(),
			Recipient:     "seed-+905551112233",
			Channel:       notification.NotificationChannelSMS,
			Content:       "Your OTP is 789012.",
			Priority:      notification.NotificationPriorityHigh,
			Status:        notification.NotificationStatusRetrying,
			RetryCount:    1,
			IdempotencyKey: &idemKey2,
		},
		{
			ID:        uuid.New(),
			Recipient: "seed-user-4@example.com",
			Channel:   notification.NotificationChannelEmail,
			Content:   "Your subscription has been cancelled.",
			Priority:  notification.NotificationPriorityNormal,
			Status:    notification.NotificationStatusCancelled,
		},
		{
			ID:        uuid.New(),
			Recipient: "seed-device-token-ghi",
			Channel:   notification.NotificationChannelPush,
			Content:   "New message from John.",
			Priority:  notification.NotificationPriorityLow,
			Status:    notification.NotificationStatusPending,
		},
	}

	if err := db.Create(&notifications).Error; err != nil {
		log.Fatalf("failed to create notifications: %v", err)
	}
	log.Printf("Created %d individual notifications", len(notifications))

	// --- Batch Notifications (5 with shared BatchID) ---

	batchID := uuid.New()

	batchNotifications := []notification.Notification{
		{
			ID:        uuid.New(),
			Recipient: "seed-batch-user-1@example.com",
			Channel:   notification.NotificationChannelEmail,
			Content:   "Monthly newsletter - March 2026 edition.",
			Priority:  notification.NotificationPriorityLow,
			Status:    notification.NotificationStatusSent,
			BatchID:   &batchID,
			SentAt:    &past,
		},
		{
			ID:        uuid.New(),
			Recipient: "seed-batch-user-2@example.com",
			Channel:   notification.NotificationChannelEmail,
			Content:   "Monthly newsletter - March 2026 edition.",
			Priority:  notification.NotificationPriorityLow,
			Status:    notification.NotificationStatusSent,
			BatchID:   &batchID,
			SentAt:    &past,
		},
		{
			ID:        uuid.New(),
			Recipient: "seed-batch-user-3@example.com",
			Channel:   notification.NotificationChannelEmail,
			Content:   "Monthly newsletter - March 2026 edition.",
			Priority:  notification.NotificationPriorityLow,
			Status:    notification.NotificationStatusPending,
			BatchID:   &batchID,
		},
		{
			ID:        uuid.New(),
			Recipient: "seed-batch-user-4@example.com",
			Channel:   notification.NotificationChannelEmail,
			Content:   "Monthly newsletter - March 2026 edition.",
			Priority:  notification.NotificationPriorityLow,
			Status:    notification.NotificationStatusPending,
			BatchID:   &batchID,
		},
		{
			ID:            uuid.New(),
			Recipient:     "seed-batch-user-5@example.com",
			Channel:       notification.NotificationChannelEmail,
			Content:       "Monthly newsletter - March 2026 edition.",
			Priority:      notification.NotificationPriorityLow,
			Status:        notification.NotificationStatusFailed,
			BatchID:       &batchID,
			FailedAt:      &past,
			FailureReason: &failureReason,
			RetryCount:    2,
		},
	}

	if err := db.Create(&batchNotifications).Error; err != nil {
		log.Fatalf("failed to create batch notifications: %v", err)
	}
	log.Printf("Created %d batch notifications (batchID=%s)", len(batchNotifications), batchID)

	fmt.Println()
	fmt.Println("=== Seed complete ===")
	fmt.Printf("  Templates:              %d\n", len(templates))
	fmt.Printf("  Individual notifications: %d\n", len(notifications))
	fmt.Printf("  Batch notifications:      %d (batchID=%s)\n", len(batchNotifications), batchID)
	fmt.Printf("  Total notifications:      %d\n", len(notifications)+len(batchNotifications))
}
