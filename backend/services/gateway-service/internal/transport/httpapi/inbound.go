package httpapi

import (
	"crypto/subtle"
	"io"
	"net/http"
	"strings"

	"order-fill/backend/pkg/grpcutil"
	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	inboundv1 "order-fill/backend/proto/gen/go/orderfill/inbound/v1"
)

const inboundWebhookLimit = 32 << 20

type inboundSettingsJSON struct {
	Enabled        bool   `json:"enabled"`
	ReceiveAddress string `json:"receive_address"`
	LastWebhookAt  string `json:"last_webhook_at"`
	WebhookCount   int64  `json:"webhook_count"`
	ErrorCount     int64  `json:"error_count"`
}

type inboundCompanyJSON struct {
	CompanyID      string `json:"company_id"`
	ReceiveAddress string `json:"receive_address"`
	SenderEmail    string `json:"sender_email"`
	Enabled        bool   `json:"enabled"`
}

type inboundAttachmentJSON struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

type inboundMessageJSON struct {
	ID                string                  `json:"id"`
	ProviderMessageID string                  `json:"provider_message_id"`
	Subject           string                  `json:"subject"`
	EnvelopeFrom      string                  `json:"envelope_from"`
	EnvelopeTo        string                  `json:"envelope_to"`
	ReceivedAt        string                  `json:"received_at"`
	CompanyID         string                  `json:"company_id"`
	Status            string                  `json:"status"`
	ErrorCode         string                  `json:"error_code"`
	AttachmentCount   int32                   `json:"attachment_count"`
	TotalBytes        int64                   `json:"total_bytes"`
	Attachments       []inboundAttachmentJSON `json:"attachments"`
}

type inboundMessageBodyJSON struct {
	inboundMessageJSON
	BodyText string `json:"body_text"`
	BodyHTML string `json:"body_html"`
}

func (a *API) inboundWebhookAuthorized(r *http.Request) bool {
	provided := strings.TrimSpace(r.Header.Get("Authorization"))
	expected := a.InboundWebhookToken
	if expected == "" {
		return false
	}
	if strings.HasPrefix(strings.ToLower(provided), "bearer ") {
		provided = strings.TrimSpace(provided[len("Bearer "):])
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

func (a *API) inboundWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, inboundWebhookLimit))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid body")
		return
	}
	if len(body) == 0 {
		writeError(w, http.StatusBadRequest, "bad_request", "empty body")
		return
	}
	ctx := grpcutil.WithWorkerToken(r.Context(), a.WorkerToken)
	_, err = a.Clients.Inbound.IngestWebhook(ctx, &inboundv1.IngestWebhookRequest{
		Meta:       &commonv1.RequestMeta{RequestId: grpcutil.NewID()},
		RawPayload: body,
	})
	if err != nil {
		writeGRPCError(w, "inbound_ingest_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *API) inboundSettings(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	if user.Role != "platform_admin" {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	resp, err := a.Clients.Inbound.GetSettings(a.jobCtx(r, user), &inboundv1.GetSettingsRequest{})
	if err != nil {
		writeGRPCError(w, "inbound_settings_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, presentInboundSettings(resp.GetSettings()))
}

func (a *API) updateInboundSettings(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	if user.Role != "platform_admin" {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	var payload struct {
		Enabled        bool   `json:"enabled"`
		ReceiveAddress string `json:"receive_address"`
	}
	if !decodeJSON(w, r, &payload, authJSONLimit) {
		return
	}
	resp, err := a.Clients.Inbound.UpdateSettings(a.jobCtx(r, user), &inboundv1.UpdateSettingsRequest{
		Meta: a.meta(user), Enabled: payload.Enabled, ReceiveAddress: payload.ReceiveAddress,
	})
	if err != nil {
		writeGRPCError(w, "inbound_settings_update_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, presentInboundSettings(resp.GetSettings()))
}

func (a *API) inboundCompany(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	companyID := r.PathValue("company_id")
	if !a.inboundCompanyReadAllowed(user, companyID) {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	resp, err := a.Clients.Inbound.GetCompanyInbound(a.jobCtx(r, user), &inboundv1.GetCompanyInboundRequest{
		Meta: a.meta(user), CompanyId: companyID,
	})
	if err != nil {
		writeGRPCError(w, "inbound_company_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, presentInboundCompany(resp.GetCompanyInbound()))
}

func (a *API) updateInboundCompany(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	companyID := r.PathValue("company_id")
	if !a.inboundCompanyReadAllowed(user, companyID) || user.Role != "company_owner" {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	var payload struct {
		ReceiveAddress string `json:"receive_address"`
		SenderEmail    string `json:"sender_email"`
		Enabled        *bool  `json:"enabled"`
	}
	if !decodeJSON(w, r, &payload, authJSONLimit) {
		return
	}
	enabled := true
	if payload.Enabled != nil {
		enabled = *payload.Enabled
	}
	resp, err := a.Clients.Inbound.UpdateCompanyInbound(a.jobCtx(r, user), &inboundv1.UpdateCompanyInboundRequest{
		Meta: a.meta(user), CompanyId: companyID,
		ReceiveAddress: payload.ReceiveAddress, SenderEmail: payload.SenderEmail, Enabled: enabled,
	})
	if err != nil {
		writeGRPCError(w, "inbound_company_update_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, presentInboundCompany(resp.GetCompanyInbound()))
}

func (a *API) inboundMessages(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	companyID := r.PathValue("company_id")
	if !a.inboundCompanyReadAllowed(user, companyID) {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	resp, err := a.Clients.Inbound.ListMessages(a.jobCtx(r, user), &inboundv1.ListMessagesRequest{
		Meta: a.meta(user), CompanyId: companyID,
	})
	if err != nil {
		writeGRPCError(w, "inbound_messages_failed", err)
		return
	}
	messages := make([]inboundMessageJSON, 0, len(resp.GetMessages()))
	for _, m := range resp.GetMessages() {
		messages = append(messages, presentInboundMessage(m))
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": messages})
}

func (a *API) inboundMessageFile(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	companyID := r.PathValue("company_id")
	if !a.inboundCompanyReadAllowed(user, companyID) {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	resp, err := a.Clients.Inbound.GetMessageFile(a.jobCtx(r, user), &inboundv1.GetMessageFileRequest{
		Meta: a.meta(user), MessageId: r.PathValue("message_id"), AttachmentId: r.PathValue("attachment_id"),
	})
	if err != nil {
		writeGRPCError(w, "inbound_message_file_failed", err)
		return
	}
	att := resp.GetAttachment()
	name := att.GetName()
	contentType := att.GetContentType()
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", contentDisposition(name))
	if _, err := w.Write(resp.GetBody()); err != nil {
		return
	}
}

func (a *API) inboundMessage(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	companyID := r.PathValue("company_id")
	if !a.inboundCompanyReadAllowed(user, companyID) {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	resp, err := a.Clients.Inbound.GetMessage(a.jobCtx(r, user), &inboundv1.GetMessageRequest{
		Meta: a.meta(user), CompanyId: companyID, MessageId: r.PathValue("message_id"),
	})
	if err != nil {
		writeGRPCError(w, "inbound_message_failed", err)
		return
	}
	message := resp.GetMessage()
	out := inboundMessageBodyJSON{
		inboundMessageJSON: presentInboundMessage(message),
		BodyText:           message.GetBodyText(),
		BodyHTML:           message.GetBodyHtml(),
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) inboundDeliveries(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	if user.Role != "platform_admin" {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	resp, err := a.Clients.Inbound.ListDeliveries(a.jobCtx(r, user), &inboundv1.ListDeliveriesRequest{
		Meta: a.meta(user),
	})
	if err != nil {
		writeGRPCError(w, "inbound_deliveries_failed", err)
		return
	}
	deliveries := make([]inboundMessageJSON, 0, len(resp.GetDeliveries()))
	for _, m := range resp.GetDeliveries() {
		item := presentInboundMessage(m)
		item.Subject = ""
		item.EnvelopeFrom = ""
		deliveries = append(deliveries, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"deliveries": deliveries})
}

func (a *API) inboundCompanyReadAllowed(user User, companyID string) bool {
	if user.CompanyID == "" || companyID == "" {
		return false
	}
	if user.CompanyID != companyID {
		return false
	}
	return user.Role == "company_owner" || user.Role == "company_admin"
}

func presentInboundSettings(in *inboundv1.InboundSettings) inboundSettingsJSON {
	if in == nil {
		return inboundSettingsJSON{}
	}
	return inboundSettingsJSON{
		Enabled:        in.GetEnabled(),
		ReceiveAddress: in.GetReceiveAddress(),
		LastWebhookAt:  in.GetLastWebhookAt(),
		WebhookCount:   in.GetWebhookCount(),
		ErrorCount:     in.GetErrorCount(),
	}
}

func presentInboundCompany(in *inboundv1.CompanyInbound) inboundCompanyJSON {
	if in == nil {
		return inboundCompanyJSON{}
	}
	return inboundCompanyJSON{
		CompanyID:      in.GetCompanyId(),
		ReceiveAddress: in.GetReceiveAddress(),
		SenderEmail:    in.GetSenderEmail(),
		Enabled:        in.GetEnabled(),
	}
}

func presentInboundMessage(in *inboundv1.InboundMessageSummary) inboundMessageJSON {
	if in == nil {
		return inboundMessageJSON{}
	}
	out := inboundMessageJSON{
		ID:                in.GetId(),
		ProviderMessageID: in.GetProviderMessageId(),
		Subject:           in.GetSubject(),
		EnvelopeFrom:      in.GetEnvelopeFrom(),
		EnvelopeTo:        in.GetEnvelopeTo(),
		ReceivedAt:        in.GetReceivedAt(),
		CompanyID:         in.GetCompanyId(),
		Status:            inboundStatusString(in.GetStatus()),
		ErrorCode:         in.GetErrorCode(),
		AttachmentCount:   in.GetAttachmentCount(),
		TotalBytes:        in.GetTotalBytes(),
	}
	if atts := in.GetAttachments(); len(atts) > 0 {
		out.Attachments = make([]inboundAttachmentJSON, len(atts))
		for i, a := range atts {
			out.Attachments[i] = inboundAttachmentJSON{
				ID:          a.GetId(),
				Name:        a.GetName(),
				ContentType: a.GetContentType(),
				Size:        a.GetSize(),
			}
		}
	}
	return out
}

func inboundStatusString(st inboundv1.InboundMessageStatus) string {
	switch st {
	case inboundv1.InboundMessageStatus_INBOUND_MESSAGE_STATUS_RECEIVED:
		return "received"
	case inboundv1.InboundMessageStatus_INBOUND_MESSAGE_STATUS_PROCESSED:
		return "processed"
	case inboundv1.InboundMessageStatus_INBOUND_MESSAGE_STATUS_ERROR_UNKNOWN_ADDRESS:
		return "error:unknown_address"
	case inboundv1.InboundMessageStatus_INBOUND_MESSAGE_STATUS_ERROR_MISMATCH_FROM:
		return "error:mismatch_from"
	case inboundv1.InboundMessageStatus_INBOUND_MESSAGE_STATUS_ERROR_NO_ATTACHMENTS:
		return "error:no_attachments"
	case inboundv1.InboundMessageStatus_INBOUND_MESSAGE_STATUS_ERROR_TOO_LARGE:
		return "error:too_large"
	default:
		return "unknown"
	}
}
