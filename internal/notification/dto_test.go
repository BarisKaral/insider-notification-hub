package notification

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func strPtr(s string) *string       { return &s }
func uuidPtr(u uuid.UUID) *uuid.UUID { return &u }
func timePtr(t time.Time) *time.Time { return &t }

// --- NotificationCreateRequest Validation Tests ---

func TestDTO_CreateRequest_ValidWithContent(t *testing.T) {
	req := NotificationCreateRequest{
		Recipient: "user@example.com",
		Channel:   "email",
		Content:   strPtr("Hello, world!"),
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestDTO_CreateRequest_ValidWithTemplate(t *testing.T) {
	req := NotificationCreateRequest{
		Recipient:  "user@example.com",
		Channel:    "sms",
		TemplateID: uuidPtr(uuid.New()),
		Variables:  map[string]string{"name": "Baris"},
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestDTO_CreateRequest_ContentXORTemplate_BothProvided(t *testing.T) {
	req := NotificationCreateRequest{
		Recipient:  "user@example.com",
		Channel:    "email",
		Content:    strPtr("Hello"),
		TemplateID: uuidPtr(uuid.New()),
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error when both content and templateId are provided")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected mutually exclusive error, got: %v", err)
	}
}

func TestDTO_CreateRequest_ContentXORTemplate_NeitherProvided(t *testing.T) {
	req := NotificationCreateRequest{
		Recipient: "user@example.com",
		Channel:   "email",
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error when neither content nor templateId is provided")
	}
	if !strings.Contains(err.Error(), "either content or templateId") {
		t.Fatalf("expected 'either content or templateId' error, got: %v", err)
	}
}

func TestDTO_CreateRequest_ContentXORTemplate_EmptyContent(t *testing.T) {
	req := NotificationCreateRequest{
		Recipient: "user@example.com",
		Channel:   "email",
		Content:   strPtr(""),
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error when content is empty string and no templateId")
	}
}

func TestDTO_CreateRequest_SMSContentLimit(t *testing.T) {
	content := strings.Repeat("a", 161)
	req := NotificationCreateRequest{
		Recipient: "+1234567890",
		Channel:   "sms",
		Content:   &content,
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for SMS content exceeding 160 chars")
	}
	if !strings.Contains(err.Error(), "160") {
		t.Fatalf("expected error mentioning 160, got: %v", err)
	}
}

func TestDTO_CreateRequest_SMSContentAtLimit(t *testing.T) {
	content := strings.Repeat("a", 160)
	req := NotificationCreateRequest{
		Recipient: "+1234567890",
		Channel:   "sms",
		Content:   &content,
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected no error for SMS content at exactly 160 chars, got: %v", err)
	}
}

func TestDTO_CreateRequest_EmailContentLimit(t *testing.T) {
	content := strings.Repeat("a", 10001)
	req := NotificationCreateRequest{
		Recipient: "user@example.com",
		Channel:   "email",
		Content:   &content,
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for email content exceeding 10000 chars")
	}
	if !strings.Contains(err.Error(), "10000") {
		t.Fatalf("expected error mentioning 10000, got: %v", err)
	}
}

func TestDTO_CreateRequest_EmailContentAtLimit(t *testing.T) {
	content := strings.Repeat("a", 10000)
	req := NotificationCreateRequest{
		Recipient: "user@example.com",
		Channel:   "email",
		Content:   &content,
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected no error for email content at exactly 10000 chars, got: %v", err)
	}
}

func TestDTO_CreateRequest_PushContentLimit(t *testing.T) {
	content := strings.Repeat("a", 257)
	req := NotificationCreateRequest{
		Recipient: "device-token-123",
		Channel:   "push",
		Content:   &content,
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for push content exceeding 256 chars")
	}
	if !strings.Contains(err.Error(), "256") {
		t.Fatalf("expected error mentioning 256, got: %v", err)
	}
}

func TestDTO_CreateRequest_PushContentAtLimit(t *testing.T) {
	content := strings.Repeat("a", 256)
	req := NotificationCreateRequest{
		Recipient: "device-token-123",
		Channel:   "push",
		Content:   &content,
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected no error for push content at exactly 256 chars, got: %v", err)
	}
}

func TestDTO_CreateRequest_DefaultPriority(t *testing.T) {
	req := NotificationCreateRequest{
		Recipient: "user@example.com",
		Channel:   "email",
		Content:   strPtr("Hello"),
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if req.Priority != "normal" {
		t.Fatalf("expected default priority 'normal', got: %q", req.Priority)
	}
}

func TestDTO_CreateRequest_InvalidChannel(t *testing.T) {
	req := NotificationCreateRequest{
		Recipient: "user@example.com",
		Channel:   "pigeon",
		Content:   strPtr("Hello"),
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for invalid channel")
	}
}

func TestDTO_CreateRequest_InvalidPriority(t *testing.T) {
	req := NotificationCreateRequest{
		Recipient: "user@example.com",
		Channel:   "email",
		Content:   strPtr("Hello"),
		Priority:  "urgent",
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for invalid priority")
	}
}

func TestDTO_CreateRequest_MissingRecipient(t *testing.T) {
	req := NotificationCreateRequest{
		Channel: "email",
		Content: strPtr("Hello"),
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for missing recipient")
	}
}

func TestDTO_CreateRequest_MissingChannel(t *testing.T) {
	req := NotificationCreateRequest{
		Recipient: "user@example.com",
		Content:   strPtr("Hello"),
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for missing channel")
	}
}

func TestDTO_CreateRequest_WithScheduledAt(t *testing.T) {
	future := time.Now().Add(1 * time.Hour)
	req := NotificationCreateRequest{
		Recipient:   "user@example.com",
		Channel:     "email",
		Content:     strPtr("Scheduled message"),
		ScheduledAt: &future,
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

// --- NotificationBatchCreateRequest Validation Tests ---

func TestDTO_BatchCreateRequest_Valid(t *testing.T) {
	req := NotificationBatchCreateRequest{
		Notifications: []NotificationCreateRequest{
			{Recipient: "a@b.com", Channel: "email", Content: strPtr("Hello")},
			{Recipient: "b@b.com", Channel: "sms", Content: strPtr("Hi")},
		},
	}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestDTO_BatchCreateRequest_Empty(t *testing.T) {
	req := NotificationBatchCreateRequest{
		Notifications: []NotificationCreateRequest{},
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for empty batch")
	}
}

func TestDTO_BatchCreateRequest_Nil(t *testing.T) {
	req := NotificationBatchCreateRequest{}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for nil batch")
	}
}

func TestDTO_BatchCreateRequest_ExceedsMax(t *testing.T) {
	notifications := make([]NotificationCreateRequest, 1001)
	for i := range notifications {
		notifications[i] = NotificationCreateRequest{
			Recipient: "user@example.com",
			Channel:   "email",
			Content:   strPtr("Hello"),
		}
	}
	req := NotificationBatchCreateRequest{Notifications: notifications}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for batch exceeding 1000")
	}
}

func TestDTO_BatchCreateRequest_AtMax(t *testing.T) {
	notifications := make([]NotificationCreateRequest, 1000)
	for i := range notifications {
		notifications[i] = NotificationCreateRequest{
			Recipient: "user@example.com",
			Channel:   "email",
			Content:   strPtr("Hello"),
		}
	}
	req := NotificationBatchCreateRequest{Notifications: notifications}
	if err := req.Validate(); err != nil {
		t.Fatalf("expected no error for batch of 1000, got: %v", err)
	}
}

func TestDTO_BatchCreateRequest_InvalidNotification(t *testing.T) {
	req := NotificationBatchCreateRequest{
		Notifications: []NotificationCreateRequest{
			{Recipient: "a@b.com", Channel: "email", Content: strPtr("OK")},
			{Recipient: "", Channel: "email", Content: strPtr("Missing recipient")},
		},
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error when a notification in the batch is invalid")
	}
	if !strings.Contains(err.Error(), "notification[1]") {
		t.Fatalf("expected error to reference notification[1], got: %v", err)
	}
}

// --- NotificationListFilter Tests ---

func TestDTO_ListFilter_Defaults(t *testing.T) {
	f := NotificationListFilter{}
	f.Normalize()
	if f.Limit != 20 {
		t.Fatalf("expected default limit 20, got: %d", f.Limit)
	}
	if f.Offset != 0 {
		t.Fatalf("expected default offset 0, got: %d", f.Offset)
	}
}

func TestDTO_ListFilter_CapsLimit(t *testing.T) {
	f := NotificationListFilter{Limit: 200}
	f.Normalize()
	if f.Limit != 100 {
		t.Fatalf("expected limit capped at 100, got: %d", f.Limit)
	}
}

func TestDTO_ListFilter_NegativeOffset(t *testing.T) {
	f := NotificationListFilter{Offset: -5}
	f.Normalize()
	if f.Offset != 0 {
		t.Fatalf("expected offset normalized to 0, got: %d", f.Offset)
	}
}

// --- ToNotificationResponse / ToNotificationResponseList Tests ---

func TestDTO_ToResponse(t *testing.T) {
	id := uuid.New()
	batchID := uuid.New()
	now := time.Now()

	n := &Notification{
		ID:            id,
		Recipient:     "user@example.com",
		Channel:       NotificationChannelEmail,
		Content:       "Hello",
		Priority:      NotificationPriorityHigh,
		Status:        NotificationStatusSent,
		BatchID:       &batchID,
		ProviderMsgID: strPtr("msg-123"),
		RetryCount:    2,
		SentAt:        &now,
		CreatedAt:     now,
	}

	resp := ToNotificationResponse(n)

	if resp.ID != id {
		t.Fatalf("expected ID %s, got %s", id, resp.ID)
	}
	if resp.Recipient != "user@example.com" {
		t.Fatalf("expected recipient 'user@example.com', got %q", resp.Recipient)
	}
	if resp.Channel != "email" {
		t.Fatalf("expected channel 'email', got %q", resp.Channel)
	}
	if resp.Content != "Hello" {
		t.Fatalf("expected content 'Hello', got %q", resp.Content)
	}
	if resp.Priority != "high" {
		t.Fatalf("expected priority 'high', got %q", resp.Priority)
	}
	if resp.Status != "sent" {
		t.Fatalf("expected status 'sent', got %q", resp.Status)
	}
	if resp.BatchID == nil || *resp.BatchID != batchID {
		t.Fatalf("expected batchID %s, got %v", batchID, resp.BatchID)
	}
	if resp.ProviderMsgID == nil || *resp.ProviderMsgID != "msg-123" {
		t.Fatalf("expected providerMsgID 'msg-123', got %v", resp.ProviderMsgID)
	}
	if resp.RetryCount != 2 {
		t.Fatalf("expected retryCount 2, got %d", resp.RetryCount)
	}
}

func TestDTO_ToResponseList(t *testing.T) {
	n1 := &Notification{ID: uuid.New(), Channel: NotificationChannelSMS, Priority: NotificationPriorityNormal, Status: NotificationStatusPending, CreatedAt: time.Now()}
	n2 := &Notification{ID: uuid.New(), Channel: NotificationChannelEmail, Priority: NotificationPriorityLow, Status: NotificationStatusQueued, CreatedAt: time.Now()}

	responses := ToNotificationResponseList([]*Notification{n1, n2})

	if len(responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(responses))
	}
	if responses[0].ID != n1.ID {
		t.Fatalf("expected first response ID %s, got %s", n1.ID, responses[0].ID)
	}
	if responses[1].ID != n2.ID {
		t.Fatalf("expected second response ID %s, got %s", n2.ID, responses[1].ID)
	}
}

func TestDTO_ToResponseList_Empty(t *testing.T) {
	responses := ToNotificationResponseList([]*Notification{})
	if len(responses) != 0 {
		t.Fatalf("expected 0 responses, got %d", len(responses))
	}
}
