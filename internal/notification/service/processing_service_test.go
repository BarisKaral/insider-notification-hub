package service

import (
	"context"
	"testing"

	"github.com/baris/notification-hub/internal/notification/domain"
	"github.com/baris/notification-hub/internal/notification/provider"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- Mocks for ProcessingService ---

type mockNotificationServiceForProcessing struct {
	mock.Mock
}

func (m *mockNotificationServiceForProcessing) Create(ctx context.Context, req domain.NotificationCreateRequest, idempotencyKey *string) (*domain.Notification, error) {
	args := m.Called(ctx, req, idempotencyKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Notification), args.Error(1)
}

func (m *mockNotificationServiceForProcessing) CreateBatch(ctx context.Context, req domain.NotificationBatchCreateRequest) ([]*domain.Notification, uuid.UUID, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Get(1).(uuid.UUID), args.Error(2)
	}
	return args.Get(0).([]*domain.Notification), args.Get(1).(uuid.UUID), args.Error(2)
}

func (m *mockNotificationServiceForProcessing) GetByID(ctx context.Context, id uuid.UUID) (*domain.Notification, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Notification), args.Error(1)
}

func (m *mockNotificationServiceForProcessing) GetByBatchID(ctx context.Context, batchID uuid.UUID) ([]*domain.Notification, error) {
	args := m.Called(ctx, batchID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Notification), args.Error(1)
}

func (m *mockNotificationServiceForProcessing) List(ctx context.Context, filter domain.NotificationListFilter) ([]*domain.Notification, int64, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]*domain.Notification), args.Get(1).(int64), args.Error(2)
}

func (m *mockNotificationServiceForProcessing) Cancel(ctx context.Context, id uuid.UUID) (*domain.Notification, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Notification), args.Error(1)
}

func (m *mockNotificationServiceForProcessing) MarkAsProcessing(ctx context.Context, id uuid.UUID) (*domain.Notification, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Notification), args.Error(1)
}

func (m *mockNotificationServiceForProcessing) MarkAsSent(ctx context.Context, id uuid.UUID, providerMsgID string) error {
	args := m.Called(ctx, id, providerMsgID)
	return args.Error(0)
}

func (m *mockNotificationServiceForProcessing) MarkAsFailed(ctx context.Context, id uuid.UUID, reason string, retryCount int) error {
	args := m.Called(ctx, id, reason, retryCount)
	return args.Error(0)
}

func (m *mockNotificationServiceForProcessing) MarkAsQueued(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockNotificationServiceForProcessing) MarkAsRetrying(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockNotificationServiceForProcessing) RecoverStuckNotifications(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *mockNotificationServiceForProcessing) PublishDueScheduled(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

type mockProvider struct {
	mock.Mock
}

func (m *mockProvider) Send(ctx context.Context, req *provider.ProviderRequest) (*provider.ProviderResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*provider.ProviderResponse), args.Error(1)
}

func newTestProcessingService(svc *mockNotificationServiceForProcessing, prod *mockNotificationProducer, providers map[domain.NotificationChannel]provider.NotificationProvider) NotificationProcessingService {
	return NewNotificationProcessingService(svc, prod, providers)
}

// --- Create Tests ---

func TestProcessingService_Create_NonScheduled_PublishesAndMarksQueued(t *testing.T) {
	svc := new(mockNotificationServiceForProcessing)
	prod := new(mockNotificationProducer)
	providers := map[domain.NotificationChannel]provider.NotificationProvider{}
	ps := newTestProcessingService(svc, prod, providers)

	content := "Hello World"
	req := domain.NotificationCreateRequest{
		Recipient: "+1234567890",
		Channel:   "sms",
		Content:   &content,
	}

	notifID := uuid.New()
	createdNotif := &domain.Notification{
		ID:       notifID,
		Status:   domain.NotificationStatusPending,
		Channel:  domain.NotificationChannelSMS,
		Content:  content,
		Priority: domain.NotificationPriorityNormal,
	}

	svc.On("Create", mock.Anything, req, (*string)(nil)).Return(createdNotif, nil)
	prod.On("Publish", mock.Anything, createdNotif).Return(nil)
	svc.On("MarkAsQueued", mock.Anything, notifID).Return(nil)

	result, err := ps.Create(context.Background(), req, nil)

	require.NoError(t, err)
	assert.Equal(t, domain.NotificationStatusQueued, result.Status)
	svc.AssertExpectations(t)
	prod.AssertExpectations(t)
}

func TestProcessingService_Create_Scheduled_NoPublish(t *testing.T) {
	svc := new(mockNotificationServiceForProcessing)
	prod := new(mockNotificationProducer)
	providers := map[domain.NotificationChannel]provider.NotificationProvider{}
	ps := newTestProcessingService(svc, prod, providers)

	content := "Scheduled"
	req := domain.NotificationCreateRequest{
		Recipient: "+1234567890",
		Channel:   "sms",
		Content:   &content,
	}

	createdNotif := &domain.Notification{
		ID:     uuid.New(),
		Status: domain.NotificationStatusScheduled,
	}

	svc.On("Create", mock.Anything, req, (*string)(nil)).Return(createdNotif, nil)

	result, err := ps.Create(context.Background(), req, nil)

	require.NoError(t, err)
	assert.Equal(t, domain.NotificationStatusScheduled, result.Status)
	prod.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything)
	svc.AssertNotCalled(t, "MarkAsQueued", mock.Anything, mock.Anything)
	svc.AssertExpectations(t)
}

func TestProcessingService_Create_PublishFails_ReturnsNotificationAsPending(t *testing.T) {
	svc := new(mockNotificationServiceForProcessing)
	prod := new(mockNotificationProducer)
	providers := map[domain.NotificationChannel]provider.NotificationProvider{}
	ps := newTestProcessingService(svc, prod, providers)

	content := "Hello"
	req := domain.NotificationCreateRequest{
		Recipient: "+1234567890",
		Channel:   "sms",
		Content:   &content,
	}

	notifID := uuid.New()
	createdNotif := &domain.Notification{
		ID:     notifID,
		Status: domain.NotificationStatusPending,
	}

	svc.On("Create", mock.Anything, req, (*string)(nil)).Return(createdNotif, nil)
	prod.On("Publish", mock.Anything, createdNotif).Return(assert.AnError)

	result, err := ps.Create(context.Background(), req, nil)

	require.NoError(t, err)
	assert.Equal(t, domain.NotificationStatusPending, result.Status)
	svc.AssertNotCalled(t, "MarkAsQueued", mock.Anything, mock.Anything)
	svc.AssertExpectations(t)
	prod.AssertExpectations(t)
}

// --- CreateBatch Tests ---

func TestProcessingService_CreateBatch_PublishesNonScheduled(t *testing.T) {
	svc := new(mockNotificationServiceForProcessing)
	prod := new(mockNotificationProducer)
	providers := map[domain.NotificationChannel]provider.NotificationProvider{}
	ps := newTestProcessingService(svc, prod, providers)

	content1 := "Hello 1"
	content2 := "Hello 2"
	req := domain.NotificationBatchCreateRequest{
		Notifications: []domain.NotificationCreateRequest{
			{Recipient: "+111", Channel: "sms", Content: &content1},
			{Recipient: "+222", Channel: "sms", Content: &content2},
		},
	}

	batchID := uuid.New()
	notifID1 := uuid.New()
	notifID2 := uuid.New()
	notifications := []*domain.Notification{
		{ID: notifID1, Status: domain.NotificationStatusPending, Channel: domain.NotificationChannelSMS, BatchID: &batchID},
		{ID: notifID2, Status: domain.NotificationStatusPending, Channel: domain.NotificationChannelSMS, BatchID: &batchID},
	}

	svc.On("CreateBatch", mock.Anything, req).Return(notifications, batchID, nil)
	prod.On("PublishBatch", mock.Anything, notifications).Return(nil)
	svc.On("MarkAsQueued", mock.Anything, notifID1).Return(nil)
	svc.On("MarkAsQueued", mock.Anything, notifID2).Return(nil)

	results, resultBatchID, err := ps.CreateBatch(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, batchID, resultBatchID)
	assert.Len(t, results, 2)
	for _, n := range results {
		assert.Equal(t, domain.NotificationStatusQueued, n.Status)
	}
	svc.AssertExpectations(t)
	prod.AssertExpectations(t)
}

func TestProcessingService_CreateBatch_MixedScheduledAndPending(t *testing.T) {
	svc := new(mockNotificationServiceForProcessing)
	prod := new(mockNotificationProducer)
	providers := map[domain.NotificationChannel]provider.NotificationProvider{}
	ps := newTestProcessingService(svc, prod, providers)

	content := "Hello"
	req := domain.NotificationBatchCreateRequest{
		Notifications: []domain.NotificationCreateRequest{
			{Recipient: "+111", Channel: "sms", Content: &content},
			{Recipient: "+222", Channel: "sms", Content: &content},
		},
	}

	batchID := uuid.New()
	notifID1 := uuid.New()
	notifID2 := uuid.New()
	notifications := []*domain.Notification{
		{ID: notifID1, Status: domain.NotificationStatusPending, Channel: domain.NotificationChannelSMS, BatchID: &batchID},
		{ID: notifID2, Status: domain.NotificationStatusScheduled, Channel: domain.NotificationChannelSMS, BatchID: &batchID},
	}

	svc.On("CreateBatch", mock.Anything, req).Return(notifications, batchID, nil)
	prod.On("PublishBatch", mock.Anything, mock.MatchedBy(func(ns []*domain.Notification) bool {
		return len(ns) == 1 && ns[0].ID == notifID1
	})).Return(nil)
	svc.On("MarkAsQueued", mock.Anything, notifID1).Return(nil)

	results, _, err := ps.CreateBatch(context.Background(), req)

	require.NoError(t, err)
	assert.Equal(t, domain.NotificationStatusQueued, results[0].Status)
	assert.Equal(t, domain.NotificationStatusScheduled, results[1].Status)
	svc.AssertNotCalled(t, "MarkAsQueued", mock.Anything, notifID2)
	svc.AssertExpectations(t)
	prod.AssertExpectations(t)
}

// --- ProcessAndSend Tests ---

func TestProcessingService_ProcessAndSend_Success(t *testing.T) {
	svc := new(mockNotificationServiceForProcessing)
	prod := new(mockNotificationProducer)
	smsProv := new(mockProvider)
	providers := map[domain.NotificationChannel]provider.NotificationProvider{
		domain.NotificationChannelSMS: smsProv,
	}
	ps := newTestProcessingService(svc, prod, providers)

	id := uuid.New()
	n := &domain.Notification{
		ID:        id,
		Recipient: "+1234567890",
		Channel:   domain.NotificationChannelSMS,
		Content:   "Hello",
		Status:    domain.NotificationStatusProcessing,
	}

	svc.On("MarkAsProcessing", mock.Anything, id).Return(n, nil)
	smsProv.On("Send", mock.Anything, &provider.ProviderRequest{
		To: "+1234567890", Channel: "sms", Content: "Hello",
	}).Return(&provider.ProviderResponse{MessageID: "prov-123"}, nil)
	svc.On("MarkAsSent", mock.Anything, id, "prov-123").Return(nil)

	result, sent, err := ps.ProcessAndSend(context.Background(), id)

	require.NoError(t, err)
	assert.True(t, sent)
	assert.Equal(t, domain.NotificationStatusSent, result.Status)
	svc.AssertExpectations(t)
	smsProv.AssertExpectations(t)
}

func TestProcessingService_ProcessAndSend_SkipsCancelled(t *testing.T) {
	svc := new(mockNotificationServiceForProcessing)
	prod := new(mockNotificationProducer)
	providers := map[domain.NotificationChannel]provider.NotificationProvider{}
	ps := newTestProcessingService(svc, prod, providers)

	id := uuid.New()
	n := &domain.Notification{
		ID:     id,
		Status: domain.NotificationStatusCancelled,
	}

	svc.On("MarkAsProcessing", mock.Anything, id).Return(n, nil)

	result, sent, err := ps.ProcessAndSend(context.Background(), id)

	require.NoError(t, err)
	assert.False(t, sent)
	assert.Equal(t, domain.NotificationStatusCancelled, result.Status)
	svc.AssertExpectations(t)
}

func TestProcessingService_ProcessAndSend_SkipsAlreadySent(t *testing.T) {
	svc := new(mockNotificationServiceForProcessing)
	prod := new(mockNotificationProducer)
	providers := map[domain.NotificationChannel]provider.NotificationProvider{}
	ps := newTestProcessingService(svc, prod, providers)

	id := uuid.New()
	n := &domain.Notification{
		ID:     id,
		Status: domain.NotificationStatusSent,
	}

	svc.On("MarkAsProcessing", mock.Anything, id).Return(n, nil)

	result, sent, err := ps.ProcessAndSend(context.Background(), id)

	require.NoError(t, err)
	assert.False(t, sent)
	assert.Equal(t, domain.NotificationStatusSent, result.Status)
	svc.AssertExpectations(t)
}

func TestProcessingService_ProcessAndSend_ProviderError(t *testing.T) {
	svc := new(mockNotificationServiceForProcessing)
	prod := new(mockNotificationProducer)
	smsProv := new(mockProvider)
	providers := map[domain.NotificationChannel]provider.NotificationProvider{
		domain.NotificationChannelSMS: smsProv,
	}
	ps := newTestProcessingService(svc, prod, providers)

	id := uuid.New()
	n := &domain.Notification{
		ID:      id,
		Channel: domain.NotificationChannelSMS,
		Status:  domain.NotificationStatusProcessing,
	}

	svc.On("MarkAsProcessing", mock.Anything, id).Return(n, nil)
	smsProv.On("Send", mock.Anything, mock.Anything).Return(nil, assert.AnError)

	result, sent, err := ps.ProcessAndSend(context.Background(), id)

	assert.Error(t, err)
	assert.False(t, sent)
	assert.Nil(t, result)
	svc.AssertExpectations(t)
	smsProv.AssertExpectations(t)
}

func TestProcessingService_ProcessAndSend_NoProviderForChannel(t *testing.T) {
	svc := new(mockNotificationServiceForProcessing)
	prod := new(mockNotificationProducer)
	providers := map[domain.NotificationChannel]provider.NotificationProvider{}
	ps := newTestProcessingService(svc, prod, providers)

	id := uuid.New()
	n := &domain.Notification{
		ID:      id,
		Channel: domain.NotificationChannelSMS,
		Status:  domain.NotificationStatusProcessing,
	}

	svc.On("MarkAsProcessing", mock.Anything, id).Return(n, nil)

	_, sent, err := ps.ProcessAndSend(context.Background(), id)

	assert.Error(t, err)
	assert.False(t, sent)
	svc.AssertExpectations(t)
}

// --- HandleDeliveryFailure Tests ---

func TestProcessingService_HandleDeliveryFailure_Retry(t *testing.T) {
	svc := new(mockNotificationServiceForProcessing)
	prod := new(mockNotificationProducer)
	providers := map[domain.NotificationChannel]provider.NotificationProvider{}
	ps := newTestProcessingService(svc, prod, providers)

	id := uuid.New()
	n := &domain.Notification{
		ID:      id,
		Channel: domain.NotificationChannelSMS,
		Status:  domain.NotificationStatusRetrying,
	}

	svc.On("MarkAsRetrying", mock.Anything, id).Return(nil)
	svc.On("GetByID", mock.Anything, id).Return(n, nil)
	prod.On("PublishToRetry", mock.Anything, n, int32(2)).Return(nil)

	result, err := ps.HandleDeliveryFailure(context.Background(), id, 1, 3)

	require.NoError(t, err)
	assert.Equal(t, domain.NotificationStatusRetrying, result.Status)
	svc.AssertExpectations(t)
	prod.AssertExpectations(t)
}

func TestProcessingService_HandleDeliveryFailure_PermanentFailure(t *testing.T) {
	svc := new(mockNotificationServiceForProcessing)
	prod := new(mockNotificationProducer)
	providers := map[domain.NotificationChannel]provider.NotificationProvider{}
	ps := newTestProcessingService(svc, prod, providers)

	id := uuid.New()
	n := &domain.Notification{
		ID:     id,
		Status: domain.NotificationStatusFailed,
	}

	svc.On("MarkAsFailed", mock.Anything, id, "max retries exceeded", 3).Return(nil)
	svc.On("GetByID", mock.Anything, id).Return(n, nil)

	result, err := ps.HandleDeliveryFailure(context.Background(), id, 3, 3)

	require.NoError(t, err)
	assert.Equal(t, domain.NotificationStatusFailed, result.Status)
	prod.AssertNotCalled(t, "PublishToRetry", mock.Anything, mock.Anything, mock.Anything)
	svc.AssertExpectations(t)
}

// --- RecoverStuckNotifications & PublishDueScheduled Tests ---

func TestProcessingService_RecoverStuckNotifications(t *testing.T) {
	svc := new(mockNotificationServiceForProcessing)
	prod := new(mockNotificationProducer)
	providers := map[domain.NotificationChannel]provider.NotificationProvider{}
	ps := newTestProcessingService(svc, prod, providers)

	svc.On("RecoverStuckNotifications", mock.Anything).Return(nil)

	err := ps.RecoverStuckNotifications(context.Background())

	require.NoError(t, err)
	svc.AssertExpectations(t)
}

func TestProcessingService_PublishDueScheduled(t *testing.T) {
	svc := new(mockNotificationServiceForProcessing)
	prod := new(mockNotificationProducer)
	providers := map[domain.NotificationChannel]provider.NotificationProvider{}
	ps := newTestProcessingService(svc, prod, providers)

	svc.On("PublishDueScheduled", mock.Anything).Return(nil)

	err := ps.PublishDueScheduled(context.Background())

	require.NoError(t, err)
	svc.AssertExpectations(t)
}
