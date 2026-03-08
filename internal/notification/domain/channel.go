package domain

type NotificationChannel string

const (
	NotificationChannelSMS   NotificationChannel = "sms"
	NotificationChannelEmail NotificationChannel = "email"
	NotificationChannelPush  NotificationChannel = "push"
)
