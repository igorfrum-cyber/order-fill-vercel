package domain

import "time"

type InboundStatus string

const (
	StatusReceived            InboundStatus = "received"
	StatusProcessed           InboundStatus = "processed"
	StatusErrorUnknownAddress InboundStatus = "error:unknown_address"
	StatusErrorMismatchFrom   InboundStatus = "error:mismatch_from"
	StatusErrorNoAttachments  InboundStatus = "error:no_attachments"
	StatusErrorTooLarge       InboundStatus = "error:too_large"
)

type Settings struct {
	Enabled        bool
	ReceiveAddress string
	LastWebhookAt  time.Time
	WebhookCount   int64
	ErrorCount     int64
}

type CompanyInbound struct {
	CompanyID      string
	ReceiveAddress string
	AllowedFrom    []string
	SenderEmail    string
	Enabled        bool
}

type MessageSummary struct {
	ID                string
	ProviderMessageID string
	Subject           string
	EnvelopeFrom      string
	EnvelopeTo        string
	ReceivedAt        time.Time
	CompanyID         string
	Status            InboundStatus
	ErrorCode         string
	AttachmentCount   int32
	TotalBytes        int64
	Attachments       []Attachment
	BodyText          string
	BodyHTML          string
}

type Attachment struct {
	ID          string
	MessageID   string
	Name        string
	ContentType string
	Size        int64
	ObjectKey   string
}
