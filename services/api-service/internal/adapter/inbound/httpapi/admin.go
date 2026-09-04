package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"order-fill/services/api-service/internal/app/port"
	"order-fill/services/api-service/internal/domain/identity"
)

type adminAPI interface {
	CreateCompany(ctx context.Context, actor identity.User, name string, loginSlug string) (identity.Company, error)
	ListCompanies(ctx context.Context, actor identity.User) ([]identity.Company, error)
	SetCompanyLoginSlug(ctx context.Context, actor identity.User, companyID string, loginSlug string) (identity.Company, error)
	UpdateCompany(ctx context.Context, actor identity.User, companyID string, name string, loginSlug string) (identity.Company, error)
	SetCompanyLogo(ctx context.Context, actor identity.User, companyID string, content []byte) (identity.Company, error)
	ClearCompanyLogo(ctx context.Context, actor identity.User, companyID string) (identity.Company, error)
	DisableCompany(ctx context.Context, actor identity.User, companyID string) error
	CreateUser(ctx context.Context, actor identity.User, companyID string, login string, role identity.Role) (identity.User, string, error)
	ListUsers(ctx context.Context, actor identity.User, companyID string) ([]identity.User, error)
	DisableUser(ctx context.Context, actor identity.User, userID string) error
	ListAudit(ctx context.Context, actor identity.User) ([]port.AuditEvent, error)
	RecordAudit(ctx context.Context, actor identity.User, action string, companyID string, jobID string)
	PublicCompanyLogin(ctx context.Context, slug string) (identity.Company, error)
	PublicCompanyLogo(ctx context.Context, slug string) (port.Object, error)
}

type lister interface {
	Execute(ctx context.Context, actor identity.User, companyID string) ([]port.JobListRow, error)
}

type statusReader interface {
	Snapshot(ctx context.Context, actor identity.User) ([]port.ComponentStatus, error)
}

type adminHandler struct {
	admin  adminAPI
	reset  accessResetter
	lister lister
	status statusReader
}

func (h adminHandler) listJobs(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	if h.lister == nil {
		writeJSON(w, http.StatusOK, map[string]any{"jobs": []any{}})
		return
	}
	rows, err := h.lister.Execute(r.Context(), user, r.URL.Query().Get("company_id"))
	if err != nil {
		writeDomainError(w, "list_jobs_failed", err)
		return
	}
	items := make([]jobListResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, presentJobList(row))
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": items})
}

func (h adminHandler) listCompanies(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	if h.admin == nil {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	companies, err := h.admin.ListCompanies(r.Context(), user)
	if err != nil {
		writeDomainError(w, "list_companies_failed", err)
		return
	}
	items := make([]companyResponse, 0, len(companies))
	for _, company := range companies {
		items = append(items, presentCompany(company))
	}
	writeJSON(w, http.StatusOK, map[string]any{"companies": items})
}

func (h adminHandler) createCompany(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	if h.admin == nil {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	var payload struct {
		Name      string `json:"name"`
		LoginSlug string `json:"login_slug"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	company, err := h.admin.CreateCompany(r.Context(), user, payload.Name, payload.LoginSlug)
	if err != nil {
		writeDomainError(w, "create_company_failed", err)
		return
	}
	writeJSON(w, http.StatusCreated, presentCompany(company))
}

func (h adminHandler) disableCompany(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	if h.admin == nil {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	if err := h.admin.DisableCompany(r.Context(), user, r.PathValue("company_id")); err != nil {
		writeDomainError(w, "disable_company_failed", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h adminHandler) setCompanyLoginSlug(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	if h.admin == nil {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	var payload struct {
		LoginSlug string `json:"login_slug"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	company, err := h.admin.SetCompanyLoginSlug(r.Context(), user, r.PathValue("company_id"), payload.LoginSlug)
	if err != nil {
		writeDomainError(w, "set_company_login_slug_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, presentCompany(company))
}

func (h adminHandler) updateCompany(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	if h.admin == nil {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	var payload struct {
		Name      string `json:"name"`
		LoginSlug string `json:"login_slug"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	company, err := h.admin.UpdateCompany(r.Context(), user, r.PathValue("company_id"), payload.Name, payload.LoginSlug)
	if err != nil {
		writeDomainError(w, "update_company_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, presentCompany(company))
}

func (h adminHandler) setCompanyLogo(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	if h.admin == nil {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, identity.LogoMaxBytes+64<<10)
	if err := r.ParseMultipartForm(identity.LogoMaxBytes + 64<<10); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid logo")
		return
	}
	file, _, err := r.FormFile("logo")
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "logo is required")
		return
	}
	defer func() { _ = file.Close() }()
	content, err := io.ReadAll(io.LimitReader(file, identity.LogoMaxBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid logo")
		return
	}
	company, err := h.admin.SetCompanyLogo(r.Context(), user, r.PathValue("company_id"), content)
	if err != nil {
		writeDomainError(w, "set_company_logo_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, presentCompany(company))
}

func (h adminHandler) clearCompanyLogo(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	if h.admin == nil {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	company, err := h.admin.ClearCompanyLogo(r.Context(), user, r.PathValue("company_id"))
	if err != nil {
		writeDomainError(w, "clear_company_logo_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, presentCompany(company))
}

func (h adminHandler) listUsers(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	if h.admin == nil {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	users, err := h.admin.ListUsers(r.Context(), user, r.PathValue("company_id"))
	if err != nil {
		writeDomainError(w, "list_users_failed", err)
		return
	}
	items := make([]userResponse, 0, len(users))
	for _, item := range users {
		items = append(items, presentUser(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": items})
}

func (h adminHandler) createUser(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	if h.admin == nil {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	var payload struct {
		Login string `json:"login"`
		Role  string `json:"role"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	created, invite, err := h.admin.CreateUser(r.Context(), user, r.PathValue("company_id"), payload.Login, identity.Role(payload.Role))
	if err != nil {
		writeDomainError(w, "create_user_failed", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"user":       presentUser(created),
		"invite_url": "/invite/" + invite,
	})
}

func (h adminHandler) disableUser(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	if h.admin == nil {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	if err := h.admin.DisableUser(r.Context(), user, r.PathValue("user_id")); err != nil {
		writeDomainError(w, "disable_user_failed", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h adminHandler) resetUser(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	if h.reset == nil {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	invite, err := h.reset.ResetAccess(r.Context(), user, r.PathValue("user_id"))
	if err != nil {
		writeDomainError(w, "reset_user_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"invite_url": "/invite/" + invite})
}

func (h adminHandler) listAudit(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	if h.admin == nil {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	events, err := h.admin.ListAudit(r.Context(), user)
	if err != nil {
		writeDomainError(w, "list_audit_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": presentAudit(events)})
}

func (h adminHandler) listStatus(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	if h.status == nil {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	items, err := h.status.Snapshot(r.Context(), user)
	if err != nil {
		writeDomainError(w, "list_status_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"components": presentStatus(items)})
}

func (h adminHandler) publicCompanyLogin(w http.ResponseWriter, r *http.Request) {
	if h.admin == nil {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	company, err := h.admin.PublicCompanyLogin(r.Context(), r.PathValue("slug"))
	if err != nil {
		writeDomainError(w, "company_login_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name":       company.Name,
		"login_slug": company.LoginSlug,
		"has_logo":   company.HasLogo(),
	})
}

func (h adminHandler) publicCompanyLogo(w http.ResponseWriter, r *http.Request) {
	if h.admin == nil {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	object, err := h.admin.PublicCompanyLogo(r.Context(), r.PathValue("slug"))
	if err != nil {
		writeDomainError(w, "company_logo_failed", err)
		return
	}
	w.Header().Set("Content-Type", object.ContentType)
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.Header().Set("Content-Length", fmt.Sprint(len(object.Content)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(object.Content)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dest any) bool {
	return decodeJSONLimited(w, r, dest, authJSONLimit)
}

func decodeJSONLimited(w http.ResponseWriter, r *http.Request, dest any, limit int64) bool {
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid json")
		return false
	}
	return true
}

type companyResponse struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	LoginSlug  string  `json:"login_slug,omitempty"`
	HasLogo    bool    `json:"has_logo"`
	CreatedAt  string  `json:"created_at"`
	DisabledAt *string `json:"disabled_at,omitempty"`
}

func presentCompany(company identity.Company) companyResponse {
	response := companyResponse{
		ID:        company.ID,
		Name:      company.Name,
		LoginSlug: company.LoginSlug,
		HasLogo:   company.HasLogo(),
		CreatedAt: company.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if company.DisabledAt != nil {
		value := company.DisabledAt.UTC().Format("2006-01-02T15:04:05Z")
		response.DisabledAt = &value
	}
	return response
}

type jobListResponse struct {
	ID             string  `json:"id"`
	Type           string  `json:"type"`
	Status         string  `json:"status"`
	Brand          string  `json:"brand,omitempty"`
	OrderMonth     string  `json:"order_month,omitempty"`
	CompanyID      string  `json:"company_id,omitempty"`
	CreatedBy      string  `json:"created_by,omitempty"`
	CreatedByLogin string  `json:"created_by_login,omitempty"`
	CreatedAt      string  `json:"created_at"`
	Progress       float64 `json:"progress"`
}

func presentJobList(row port.JobListRow) jobListResponse {
	return jobListResponse{
		ID:             row.Job.ID,
		Type:           string(row.Job.Type),
		Status:         string(row.Job.Status),
		Brand:          row.Job.Brand,
		OrderMonth:     row.Job.OrderMonth,
		CompanyID:      row.Job.CompanyID,
		CreatedBy:      row.Job.CreatedBy,
		CreatedByLogin: row.CreatedByLogin,
		CreatedAt:      row.Job.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		Progress:       row.Job.Progress,
	}
}

type auditResponse struct {
	ID          string `json:"id"`
	At          string `json:"at"`
	ActorID     string `json:"actor_id"`
	ActorLogin  string `json:"actor_login,omitempty"`
	Action      string `json:"action"`
	CompanyID   string `json:"company_id,omitempty"`
	CompanyName string `json:"company_name,omitempty"`
	JobID       string `json:"job_id,omitempty"`
}

func presentAudit(events []port.AuditEvent) []auditResponse {
	items := make([]auditResponse, 0, len(events))
	for _, event := range events {
		items = append(items, auditResponse{
			ID:          event.ID,
			At:          event.At.UTC().Format("2006-01-02T15:04:05Z"),
			ActorID:     event.ActorID,
			ActorLogin:  event.ActorLogin,
			Action:      event.Action,
			CompanyID:   event.CompanyID,
			CompanyName: event.CompanyName,
			JobID:       event.JobID,
		})
	}
	return items
}

type statusResponse struct {
	ID string `json:"id"`
	OK bool   `json:"ok"`
}

func presentStatus(items []port.ComponentStatus) []statusResponse {
	out := make([]statusResponse, 0, len(items))
	for _, item := range items {
		out = append(out, statusResponse{ID: item.ID, OK: item.OK})
	}
	return out
}
