package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"order-fill/backend/services/inbound-service/internal/domain"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) GetSettings(ctx context.Context) (domain.Settings, error) {
	row := s.pool.QueryRow(ctx, "SELECT enabled, last_webhook_at, webhook_count, error_count FROM inbound_settings LIMIT 1")
	var out domain.Settings
	var last sql.NullTime
	err := row.Scan(&out.Enabled, &last, &out.WebhookCount, &out.ErrorCount)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Settings{Enabled: true}, nil
		}
		return domain.Settings{}, fmt.Errorf("get inbound settings: %w", err)
	}
	if last.Valid {
		out.LastWebhookAt = last.Time
	}
	return out, nil
}

func (s *Store) UpsertSettings(ctx context.Context, enabled bool) (domain.Settings, error) {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO inbound_settings (enabled) VALUES ($1)
		ON CONFLICT () DO UPDATE SET enabled = $1
	`, enabled)
	if err != nil {
		return domain.Settings{}, fmt.Errorf("upsert inbound settings: %w", err)
	}
	return s.GetSettings(ctx)
}

func (s *Store) IncrWebhookCount(ctx context.Context, isError bool) {
	col := "webhook_count"
	if isError {
		col = "error_count"
	}
	_, _ = s.pool.Exec(ctx, fmt.Sprintf(`
		INSERT INTO inbound_settings (enabled, %s) VALUES (TRUE, 1)
		ON CONFLICT () DO UPDATE SET %s = inbound_settings.%s + 1, last_webhook_at = now()
	`, col, col, col))
}

func (s *Store) GetCompanyInbound(ctx context.Context, companyID string) (domain.CompanyInbound, error) {
	row := s.pool.QueryRow(ctx, "SELECT company_id, receive_address, allowed_from, enabled FROM company_inbound WHERE company_id = $1", companyID)
	var out domain.CompanyInbound
	err := row.Scan(&out.CompanyID, &out.ReceiveAddress, &out.AllowedFrom, &out.Enabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.CompanyInbound{}, domain.ErrNotFound
		}
		return domain.CompanyInbound{}, fmt.Errorf("get company inbound: %w", err)
	}
	return out, nil
}

func (s *Store) GetCompanyInboundByAddress(ctx context.Context, address string) (domain.CompanyInbound, error) {
	addr := strings.TrimSpace(strings.ToLower(address))
	row := s.pool.QueryRow(ctx, "SELECT company_id, receive_address, allowed_from, enabled FROM company_inbound WHERE LOWER(receive_address) = $1 AND enabled = TRUE", addr)
	var out domain.CompanyInbound
	err := row.Scan(&out.CompanyID, &out.ReceiveAddress, &out.AllowedFrom, &out.Enabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.CompanyInbound{}, domain.ErrNotFound
		}
		return domain.CompanyInbound{}, fmt.Errorf("get company inbound by address: %w", err)
	}
	return out, nil
}

func (s *Store) UpsertCompanyInbound(ctx context.Context, companyID, address string, allowedFrom []string, enabled bool) (domain.CompanyInbound, error) {
	if allowedFrom == nil {
		allowedFrom = []string{}
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO company_inbound (company_id, receive_address, allowed_from, enabled)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (company_id) DO UPDATE
		SET receive_address = $2, allowed_from = $3, enabled = $4
	`, companyID, strings.TrimSpace(address), allowedFrom, enabled)
	if err != nil {
		return domain.CompanyInbound{}, fmt.Errorf("upsert company inbound: %w", err)
	}
	return s.GetCompanyInbound(ctx, companyID)
}

func (s *Store) MessageExists(ctx context.Context, providerMessageID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM inbound_message WHERE provider_message_id = $1)", providerMessageID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check message exists: %w", err)
	}
	return exists, nil
}

func (s *Store) SaveMessage(ctx context.Context, msg domain.MessageSummary, attachments []domain.Attachment) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin save message: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	now := time.Now().UTC()
	_, err = tx.Exec(ctx, `
		INSERT INTO inbound_message (id, provider_message_id, envelope_from, envelope_to, received_at, company_id, status, error_code, attachment_count, total_bytes)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
	`, msg.ID, msg.ProviderMessageID, msg.EnvelopeFrom, msg.EnvelopeTo, now, msg.CompanyID, string(msg.Status), msg.ErrorCode, msg.AttachmentCount, msg.TotalBytes)
	if err != nil {
		return fmt.Errorf("insert inbound message: %w", err)
	}

	for _, a := range attachments {
		_, err = tx.Exec(ctx, `
			INSERT INTO inbound_attachment (id, message_id, name, content_type, size, object_key)
			VALUES ($1,$2,$3,$4,$5,$6)
		`, a.ID, a.MessageID, a.Name, a.ContentType, a.Size, a.ObjectKey)
		if err != nil {
			return fmt.Errorf("insert inbound attachment: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (s *Store) ListMessages(ctx context.Context, companyID string, limit int) ([]domain.MessageSummary, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, provider_message_id, envelope_from, envelope_to, received_at, company_id, status, error_code, attachment_count, total_bytes
		FROM inbound_message
		WHERE company_id = $1
		ORDER BY received_at DESC
		LIMIT $2
	`, companyID, limit)
	if err != nil {
		return nil, fmt.Errorf("list inbound messages: %w", err)
	}
	defer rows.Close()

	var out []domain.MessageSummary
	for rows.Next() {
		var m domain.MessageSummary
		if err := rows.Scan(&m.ID, &m.ProviderMessageID, &m.EnvelopeFrom, &m.EnvelopeTo, &m.ReceivedAt, &m.CompanyID, &m.Status, &m.ErrorCode, &m.AttachmentCount, &m.TotalBytes); err != nil {
			return nil, fmt.Errorf("scan inbound message: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) GetMessageAttachments(ctx context.Context, messageID string) ([]domain.Attachment, error) {
	rows, err := s.pool.Query(ctx, "SELECT id, message_id, name, content_type, size, object_key FROM inbound_attachment WHERE message_id = $1 ORDER BY name", messageID)
	if err != nil {
		return nil, fmt.Errorf("list inbound attachments: %w", err)
	}
	defer rows.Close()

	var out []domain.Attachment
	for rows.Next() {
		var a domain.Attachment
		if err := rows.Scan(&a.ID, &a.MessageID, &a.Name, &a.ContentType, &a.Size, &a.ObjectKey); err != nil {
			return nil, fmt.Errorf("scan inbound attachment: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) GetAttachment(ctx context.Context, attachmentID string) (domain.Attachment, error) {
	row := s.pool.QueryRow(ctx, "SELECT id, message_id, name, content_type, size, object_key FROM inbound_attachment WHERE id = $1", attachmentID)
	var a domain.Attachment
	err := row.Scan(&a.ID, &a.MessageID, &a.Name, &a.ContentType, &a.Size, &a.ObjectKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Attachment{}, domain.ErrNotFound
		}
		return domain.Attachment{}, fmt.Errorf("get inbound attachment: %w", err)
	}
	return a, nil
}

func (s *Store) GetMessageCompany(ctx context.Context, messageID string) (string, error) {
	var companyID string
	err := s.pool.QueryRow(ctx, "SELECT company_id FROM inbound_message WHERE id = $1", messageID).Scan(&companyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", domain.ErrNotFound
		}
		return "", fmt.Errorf("get message company: %w", err)
	}
	return companyID, nil
}

func (s *Store) ListDeliveries(ctx context.Context, limit int) ([]domain.MessageSummary, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, provider_message_id, envelope_from, envelope_to, received_at, company_id, status, error_code, attachment_count, total_bytes
		FROM inbound_message
		ORDER BY received_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list inbound deliveries: %w", err)
	}
	defer rows.Close()

	var out []domain.MessageSummary
	for rows.Next() {
		var m domain.MessageSummary
		if err := rows.Scan(&m.ID, &m.ProviderMessageID, &m.EnvelopeFrom, &m.EnvelopeTo, &m.ReceivedAt, &m.CompanyID, &m.Status, &m.ErrorCode, &m.AttachmentCount, &m.TotalBytes); err != nil {
			return nil, fmt.Errorf("scan inbound delivery: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
