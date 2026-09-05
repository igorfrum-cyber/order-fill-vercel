package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"

	auditv1 "order-fill/backend/proto/gen/go/orderfill/audit/v1"
	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	filesv1 "order-fill/backend/proto/gen/go/orderfill/files/v1"
	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
)

func (a *API) listCompanies(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	if user.Role != "platform_admin" {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	resp, err := a.Clients.Identity.ListCompanies(r.Context(), &identityv1.ListCompaniesRequest{Meta: a.meta(user)})
	if err != nil {
		writeGRPCError(w, "list_companies_failed", err)
		return
	}
	items := make([]map[string]any, 0, len(resp.GetCompanies()))
	for _, company := range resp.GetCompanies() {
		items = append(items, a.presentCompany(r.Context(), company))
	}
	writeJSON(w, http.StatusOK, map[string]any{"companies": items})
}

func (a *API) createCompany(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	var payload struct {
		Name         string `json:"name"`
		LoginSlug    string `json:"login_slug"`
		MatchingMode string `json:"matching_mode"`
	}
	if !decodeJSON(w, r, &payload, authJSONLimit) {
		return
	}
	resp, err := a.Clients.Identity.CreateCompany(r.Context(), &identityv1.CreateCompanyRequest{
		Meta: a.meta(user), Name: payload.Name, LoginSlug: payload.LoginSlug, MatchingMode: protoMatchingMode(payload.MatchingMode),
	})
	if err != nil {
		writeGRPCError(w, "create_company_failed", err)
		return
	}
	writeJSON(w, http.StatusCreated, a.presentCompany(r.Context(), resp.GetCompany()))
}

func (a *API) updateCompany(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	var payload struct {
		Name         string `json:"name"`
		LoginSlug    string `json:"login_slug"`
		MatchingMode string `json:"matching_mode"`
	}
	if !decodeJSON(w, r, &payload, authJSONLimit) {
		return
	}
	resp, err := a.Clients.Identity.UpdateCompany(r.Context(), &identityv1.UpdateCompanyRequest{
		Meta: a.meta(user), CompanyId: r.PathValue("company_id"), Name: payload.Name, LoginSlug: payload.LoginSlug,
		MatchingMode: protoMatchingMode(payload.MatchingMode),
	})
	if err != nil {
		writeGRPCError(w, "update_company_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, a.presentCompanyFor(r.Context(), user, resp.GetCompany()))
}

func (a *API) disableCompany(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	companyID := r.PathValue("company_id")
	_, err := a.Clients.Identity.DisableCompany(r.Context(), &identityv1.DisableCompanyRequest{
		Meta: a.meta(user), CompanyId: companyID,
	})
	if err != nil {
		writeGRPCError(w, "disable_company_failed", err)
		return
	}
	a.recordAudit(r.Context(), user, "company_disabled", companyID, "")
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) setCompanyLogo(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	companyID := r.PathValue("company_id")
	if !canManageCompany(user, companyID) {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, logoMaxBytes+64<<10)
	if err := r.ParseMultipartForm(logoMaxBytes + 64<<10); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid logo")
		return
	}
	file, _, err := r.FormFile("logo")
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "logo is required")
		return
	}
	defer func() { _ = file.Close() }()
	content, err := io.ReadAll(io.LimitReader(file, logoMaxBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid logo")
		return
	}
	contentType, err := parseLogo(content)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	_, err = a.Clients.Files.PutObject(r.Context(), &filesv1.PutObjectRequest{
		Key: companyLogoKey(companyID), Name: "logo", ContentType: contentType, Body: content,
	})
	if err != nil {
		writeGRPCError(w, "set_company_logo_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, a.companyJSON(r.Context(), user, companyID))
}

func (a *API) clearCompanyLogo(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	companyID := r.PathValue("company_id")
	if !canManageCompany(user, companyID) {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	_, err := a.Clients.Files.PutObject(r.Context(), &filesv1.PutObjectRequest{
		Key: companyLogoKey(companyID), Name: "logo", ContentType: "application/octet-stream",
	})
	if err != nil {
		writeGRPCError(w, "clear_company_logo_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, a.companyJSON(r.Context(), user, companyID))
}

func (a *API) listUsers(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	resp, err := a.Clients.Identity.ListUsers(r.Context(), &identityv1.ListUsersRequest{
		Meta: a.meta(user), CompanyId: r.PathValue("company_id"),
	})
	if err != nil {
		writeGRPCError(w, "list_users_failed", err)
		return
	}
	items := make([]map[string]any, 0, len(resp.GetUsers()))
	for _, item := range resp.GetUsers() {
		items = append(items, a.presentUser(r.Context(), userFromProto(item)))
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": items})
}

func (a *API) createUser(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	var payload struct {
		Login string `json:"login"`
		Role  string `json:"role"`
	}
	if !decodeJSON(w, r, &payload, authJSONLimit) {
		return
	}
	companyID := r.PathValue("company_id")
	resp, err := a.Clients.Identity.CreateUser(r.Context(), &identityv1.CreateUserRequest{
		Meta: a.meta(user), CompanyId: companyID, Login: payload.Login, Role: payload.Role,
	})
	if err != nil {
		writeGRPCError(w, "create_user_failed", err)
		return
	}
	a.recordAudit(r.Context(), user, "invite_created", companyID, user.CompanyName)
	writeJSON(w, http.StatusCreated, map[string]any{
		"user":         a.presentUser(r.Context(), userFromProto(resp.GetUser())),
		"invite_token": resp.GetInviteToken(),
		"invite_url":   "/invite/" + resp.GetInviteToken(),
	})
}

func (a *API) disableUser(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	_, err := a.Clients.Identity.DisableUser(r.Context(), &identityv1.DisableUserRequest{Meta: a.meta(user), UserId: r.PathValue("user_id")})
	if err != nil {
		writeGRPCError(w, "disable_user_failed", err)
		return
	}
	a.recordAudit(r.Context(), user, "user_disabled", user.CompanyID, user.CompanyName)
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) resetUser(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	resp, err := a.Clients.Identity.ResetUserAccess(r.Context(), &identityv1.ResetUserAccessRequest{Meta: a.meta(user), UserId: r.PathValue("user_id")})
	if err != nil {
		writeGRPCError(w, "reset_user_failed", err)
		return
	}
	a.recordAudit(r.Context(), user, "access_reset", user.CompanyID, user.CompanyName)
	writeJSON(w, http.StatusOK, map[string]any{
		"invite_token": resp.GetInviteToken(),
		"invite_url":   "/invite/" + resp.GetInviteToken(),
	})
}

func (a *API) listAudit(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	resp, err := a.Clients.Audit.ListEvents(r.Context(), &auditv1.ListEventsRequest{Meta: a.meta(user), CompanyId: user.CompanyID})
	if err != nil {
		writeGRPCError(w, "list_audit_failed", err)
		return
	}
	items := make([]map[string]any, 0, len(resp.GetEvents()))
	for _, event := range resp.GetEvents() {
		items = append(items, presentAudit(event))
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": items})
}

func (a *API) publicCompanyLogin(w http.ResponseWriter, r *http.Request) {
	company, err := a.publicCompany(r.Context(), r.PathValue("slug"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name": company.Name, "login_slug": company.LoginSlug, "has_logo": a.companyHasLogo(r.Context(), company.ID),
	})
}

func (a *API) publicCompanyLogo(w http.ResponseWriter, r *http.Request) {
	company, err := a.publicCompany(r.Context(), r.PathValue("slug"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	resp, err := a.Clients.Files.GetObject(r.Context(), &filesv1.GetObjectRequest{Key: companyLogoKey(company.ID)})
	if err != nil || len(resp.GetBody()) == 0 {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	contentType := resp.GetObject().GetContentType()
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = "image/png"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp.GetBody())
}

type publicCompany struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	LoginSlug string `json:"login_slug"`
}

func (a *API) publicCompany(ctx context.Context, slug string) (publicCompany, error) {
	client := a.HTTP
	if client == nil {
		client = &http.Client{}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(a.IdentityHTTP, "/")+"/public/companies/"+url.PathEscape(slug), nil)
	if err != nil {
		return publicCompany{}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return publicCompany{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
		return publicCompany{}, errors.New("not found")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return publicCompany{}, err
	}
	var company publicCompany
	if err := json.Unmarshal(body, &company); err != nil {
		return publicCompany{}, err
	}
	return company, nil
}

func (a *API) companyHasLogo(ctx context.Context, companyID string) bool {
	if a.Clients.Files == nil || companyID == "" {
		return false
	}
	resp, err := a.Clients.Files.GetObject(ctx, &filesv1.GetObjectRequest{Key: companyLogoKey(companyID)})
	return err == nil && len(resp.GetBody()) > 0
}

func (a *API) presentCompany(ctx context.Context, c *identityv1.Company) map[string]any {
	out := presentCompany(c)
	if c != nil {
		out["has_logo"] = a.companyHasLogo(ctx, c.GetId())
	}
	return out
}

func (a *API) presentCompanyFor(ctx context.Context, user User, c *identityv1.Company) map[string]any {
	out := a.presentCompany(ctx, c)
	if user.Role != "platform_admin" {
		delete(out, "matching_mode")
	}
	return out
}

func (a *API) companyJSON(ctx context.Context, user User, companyID string) map[string]any {
	if user.Role == "platform_admin" && a.Clients.Identity != nil {
		resp, err := a.Clients.Identity.ListCompanies(ctx, &identityv1.ListCompaniesRequest{Meta: a.meta(user)})
		if err == nil {
			for _, company := range resp.GetCompanies() {
				if company.GetId() == companyID {
					return a.presentCompany(ctx, company)
				}
			}
		}
	}
	return map[string]any{
		"id": companyID, "name": user.CompanyName, "login_slug": user.LoginSlug,
		"has_logo": a.companyHasLogo(ctx, companyID),
	}
}

func (a *API) recordAudit(ctx context.Context, user User, action, companyID, companyName string) {
	if a.Clients.Audit == nil {
		return
	}
	payload, _ := json.Marshal(map[string]string{"actor_login": user.Login, "company_name": companyName})
	meta := a.meta(user)
	if companyID != "" {
		meta.CompanyId = companyID
	}
	_, _ = a.Clients.Audit.Record(ctx, &auditv1.RecordRequest{Meta: meta, Type: action, PayloadJson: string(payload)})
}

func presentAudit(event *auditv1.Event) map[string]any {
	var extra struct {
		ActorLogin  string `json:"actor_login"`
		CompanyName string `json:"company_name"`
	}
	_ = json.Unmarshal([]byte(event.GetPayloadJson()), &extra)
	return map[string]any{
		"id": event.GetId(), "at": event.GetCreatedAt(), "actor_id": event.GetActorUserId(),
		"actor_login": extra.ActorLogin, "action": event.GetType(), "company_id": event.GetCompanyId(),
		"company_name": extra.CompanyName, "job_id": event.GetJobId(),
	}
}

func presentCompany(c *identityv1.Company) map[string]any {
	if c == nil {
		return map[string]any{}
	}
	return map[string]any{
		"id": c.GetId(), "name": c.GetName(), "login_slug": c.GetLoginSlug(),
		"has_logo": c.GetHasLogo(), "created_at": c.GetCreatedAt(), "disabled_at": c.GetDisabledAt(),
		"matching_mode": companyMatchingMode(c),
	}
}

func companyMatchingMode(c *identityv1.Company) string {
	if c != nil && c.GetMatchingMode() == commonv1.MatchingMode_MATCHING_MODE_SMART {
		return "smart"
	}
	return "standard"
}

func protoMatchingMode(raw string) commonv1.MatchingMode {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "smart":
		return commonv1.MatchingMode_MATCHING_MODE_SMART
	case "standard":
		return commonv1.MatchingMode_MATCHING_MODE_STANDARD
	default:
		return commonv1.MatchingMode_MATCHING_MODE_UNSPECIFIED
	}
}
