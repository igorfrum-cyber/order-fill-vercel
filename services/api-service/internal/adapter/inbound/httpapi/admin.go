package httpapi

import (
	"context"
	"encoding/json"
	"net/http"

	"order-fill/services/api-service/internal/app/port"
	"order-fill/services/api-service/internal/domain/identity"
)

type adminAPI interface {
	CreateCompany(ctx context.Context, actor identity.User, name string) (identity.Company, error)
	ListCompanies(ctx context.Context, actor identity.User) ([]identity.Company, error)
	DisableCompany(ctx context.Context, actor identity.User, companyID string) error
	CreateUser(ctx context.Context, actor identity.User, companyID string, login string, role identity.Role) (identity.User, string, error)
	ListUsers(ctx context.Context, actor identity.User, companyID string) ([]identity.User, error)
	DisableUser(ctx context.Context, actor identity.User, userID string) error
	ListAudit(ctx context.Context, actor identity.User) ([]port.AuditEvent, error)
	RecordAudit(ctx context.Context, actor identity.User, action string, companyID string, jobID string)
}

type lister interface {
	Execute(ctx context.Context, actor identity.User, companyID string) ([]port.JobListRow, error)
}

type adminHandler struct {
	admin  adminAPI
	reset  accessResetter
	lister lister
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
		Name string `json:"name"`
	}
	if !decodeJSON(w, r, &payload) {
		return
	}
	company, err := h.admin.CreateCompany(r.Context(), user, payload.Name)
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

func decodeJSON(w http.ResponseWriter, r *http.Request, dest any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, authJSONLimit)
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid json")
		return false
	}
	return true
}

type companyResponse struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	CreatedAt  string  `json:"created_at"`
	DisabledAt *string `json:"disabled_at,omitempty"`
}

func presentCompany(company identity.Company) companyResponse {
	response := companyResponse{ID: company.ID, Name: company.Name, CreatedAt: company.CreatedAt.UTC().Format("2006-01-02T15:04:05Z")}
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
	ID        string `json:"id"`
	At        string `json:"at"`
	ActorID   string `json:"actor_id"`
	Action    string `json:"action"`
	CompanyID string `json:"company_id,omitempty"`
	JobID     string `json:"job_id,omitempty"`
}

func presentAudit(events []port.AuditEvent) []auditResponse {
	items := make([]auditResponse, 0, len(events))
	for _, event := range events {
		items = append(items, auditResponse{
			ID:        event.ID,
			At:        event.At.UTC().Format("2006-01-02T15:04:05Z"),
			ActorID:   event.ActorID,
			Action:    event.Action,
			CompanyID: event.CompanyID,
			JobID:     event.JobID,
		})
	}
	return items
}
