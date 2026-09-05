package httpapi

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"order-fill/backend/pkg/grpcutil"
	filesv1 "order-fill/backend/proto/gen/go/orderfill/files/v1"
	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
	jobsv1 "order-fill/backend/proto/gen/go/orderfill/jobs/v1"
	"order-fill/backend/services/gateway-service/internal/preview"
)

func (a *API) createOrderFill(w http.ResponseWriter, r *http.Request) {
	a.createJob(w, r, "order_fill")
}

func (a *API) createNorthMerge(w http.ResponseWriter, r *http.Request) {
	a.createJob(w, r, "north_merge")
}

func (a *API) createJob(w http.ResponseWriter, r *http.Request, jobType string) {
	user, ok := userFrom(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	if user.CompanyID == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "company_id is required")
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	ids := make([]string, 0)
	if jobType == "order_fill" {
		uploaded, err := a.uploadParts(r, "source_file", "source", true)
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		ids = append(ids, uploaded...)
	} else {
		uploaded, err := a.uploadParts(r, "tyumen_source_file", "source", false)
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		ids = append(ids, uploaded...)
	}
	blanks, err := a.uploadParts(r, "blank_files", "blank", true)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	ids = append(ids, blanks...)
	resp, err := a.Clients.Jobs.CreateJob(a.jobCtx(r, user), &jobsv1.CreateJobRequest{
		Meta: a.meta(user), Type: jobType, InputFileIds: ids, Brand: r.FormValue("brand"),
	})
	if err != nil {
		writeGRPCError(w, "create_job_failed", err)
		return
	}
	writeJSON(w, http.StatusAccepted, a.presentJob(r.Context(), user, resp.GetJob()))
}

func (a *API) uploadParts(r *http.Request, field, role string, required bool) ([]string, error) {
	if r.MultipartForm == nil {
		return nil, fmt.Errorf("multipart form is required")
	}
	headers := r.MultipartForm.File[field]
	if len(headers) == 0 {
		if required {
			return nil, fmt.Errorf("%s is required", field)
		}
		return nil, nil
	}
	ids := make([]string, 0, len(headers))
	for _, header := range headers {
		id, err := a.putUpload(r, header, role)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (a *API) putUpload(r *http.Request, header *multipart.FileHeader, role string) (string, error) {
	file, err := header.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()
	body, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	key := role + "/" + grpcutil.NewID() + "/" + header.Filename
	resp, err := a.Clients.Files.PutObject(r.Context(), &filesv1.PutObjectRequest{
		Key: key, Name: header.Filename, ContentType: header.Header.Get("Content-Type"), Body: body,
	})
	if err != nil {
		return "", err
	}
	return resp.GetObject().GetId(), nil
}

func (a *API) getJob(w http.ResponseWriter, r *http.Request) {
	user, job, ok := a.loadJob(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, a.withOwnerLogin(r.Context(), user, job, a.presentJob(r.Context(), user, job)))
}

func (a *API) listJobs(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return
	}
	resp, err := a.Clients.Jobs.ListJobs(a.jobCtx(r, user), &jobsv1.ListJobsRequest{Meta: a.meta(user)})
	if err != nil {
		writeGRPCError(w, "list_jobs_failed", err)
		return
	}
	items := make([]map[string]any, 0, len(resp.GetJobs()))
	wantCompany := strings.TrimSpace(r.URL.Query().Get("company_id"))
	jobs := make([]*jobsv1.Job, 0, len(resp.GetJobs()))
	for _, job := range resp.GetJobs() {
		if wantCompany != "" && job.GetCompanyId() != wantCompany {
			continue
		}
		jobs = append(jobs, job)
	}
	logins := a.ownerLogins(r.Context(), user, jobs)
	for _, job := range jobs {
		item := a.presentJob(r.Context(), user, job)
		item["created_by_login"] = logins[job.GetOwnerUserId()]
		items = append(items, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": items})
}

func (a *API) getReport(w http.ResponseWriter, r *http.Request) {
	_, job, ok := a.loadJob(w, r)
	if !ok {
		return
	}
	raw, err := a.getObjectByKey(r, "jobs/"+job.GetId()+"/report.json")
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "report was not found")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func (a *API) submitEdits(w http.ResponseWriter, r *http.Request) {
	user, job, ok := a.loadJob(w, r)
	if !ok {
		return
	}
	var payload struct {
		Edits []struct {
			Key, Value, Comment string
		} `json:"edits"`
	}
	if !decodeJSON(w, r, &payload, jobJSONLimit) {
		return
	}
	edits := make([]*jobsv1.Edit, 0, len(payload.Edits))
	for _, edit := range payload.Edits {
		edits = append(edits, &jobsv1.Edit{RowKey: edit.Key, Value: edit.Value, Comment: edit.Comment})
	}
	resp, err := a.Clients.Jobs.SubmitEdits(a.jobCtx(r, user), &jobsv1.SubmitEditsRequest{
		Meta: a.meta(user), JobId: job.GetId(), Edits: edits,
	})
	if err != nil {
		writeGRPCError(w, "submit_edits_failed", err)
		return
	}
	writeJSON(w, http.StatusAccepted, a.presentJob(r.Context(), user, resp.GetJob()))
}

func (a *API) listFiles(w http.ResponseWriter, r *http.Request) {
	user, job, ok := a.loadJob(w, r)
	if !ok {
		return
	}
	files, err := a.Clients.Jobs.ListFiles(a.jobCtx(r, user), &jobsv1.ListFilesRequest{Meta: a.meta(user), JobId: job.GetId()})
	if err != nil {
		writeGRPCError(w, "read_files_failed", err)
		return
	}
	out := make([]map[string]any, 0)
	for _, file := range files.GetFiles() {
		if !strings.Contains(file.GetObjectKey(), "/outputs/") && file.GetKind() != "" && !strings.Contains(strings.ToLower(file.GetKind()), "скачать") {
			continue
		}
		if strings.Contains(file.GetObjectKey(), "/outputs/") || strings.Contains(strings.ToLower(file.GetKind()), "скачать") {
			out = append(out, presentOutputFile(job.GetId(), file))
		}
	}
	if len(out) == 0 {
		for _, file := range files.GetFiles() {
			if strings.HasPrefix(file.GetId(), "output-") || strings.Contains(file.GetObjectKey(), "/outputs/") {
				out = append(out, presentOutputFile(job.GetId(), file))
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"files": out})
}

func presentOutputFile(jobID string, file *jobsv1.FileRef) map[string]any {
	label := file.GetKind()
	if label == "" || label == "source" || label == "blank" {
		label = file.GetName()
	}
	return map[string]any{
		"id": file.GetId(), "label": label, "name": file.GetName(),
		"content_type": file.GetContentType(), "size_bytes": 0,
		"download_path": "/api/v1/jobs/" + jobID + "/files/" + file.GetId(),
	}
}

func (a *API) downloadFile(w http.ResponseWriter, r *http.Request) {
	user, job, ok := a.loadJob(w, r)
	if !ok {
		return
	}
	files, err := a.Clients.Jobs.ListFiles(a.jobCtx(r, user), &jobsv1.ListFilesRequest{Meta: a.meta(user), JobId: job.GetId()})
	if err != nil {
		writeGRPCError(w, "download_failed", err)
		return
	}
	fileID := r.PathValue("file_id")
	for _, file := range files.GetFiles() {
		if file.GetId() != fileID {
			continue
		}
		obj, err := a.Clients.Files.GetObject(r.Context(), &filesv1.GetObjectRequest{Key: file.GetObjectKey()})
		if err != nil {
			writeGRPCError(w, "download_failed", err)
			return
		}
		w.Header().Set("Content-Type", obj.GetObject().GetContentType())
		w.Header().Set("Content-Disposition", contentDisposition(file.GetName()))
		w.Header().Set("Content-Length", fmt.Sprint(len(obj.GetBody())))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(obj.GetBody())
		return
	}
	writeError(w, http.StatusNotFound, "not_found", "file was not found")
}

func (a *API) downloadArchive(w http.ResponseWriter, r *http.Request) {
	user, job, ok := a.loadJob(w, r)
	if !ok {
		return
	}
	files, err := a.Clients.Jobs.ListFiles(a.jobCtx(r, user), &jobsv1.ListFilesRequest{Meta: a.meta(user), JobId: job.GetId()})
	if err != nil {
		writeGRPCError(w, "download_archive_failed", err)
		return
	}
	var buf bytes.Buffer
	zipw := zip.NewWriter(&buf)
	for _, file := range files.GetFiles() {
		if !strings.Contains(file.GetObjectKey(), "/outputs/") && !strings.HasPrefix(file.GetId(), "output-") {
			continue
		}
		obj, err := a.Clients.Files.GetObject(r.Context(), &filesv1.GetObjectRequest{Key: file.GetObjectKey()})
		if err != nil {
			writeGRPCError(w, "download_archive_failed", err)
			return
		}
		entry, err := zipw.Create(file.GetName())
		if err != nil {
			writeError(w, http.StatusInternalServerError, "download_archive_failed", err.Error())
			return
		}
		if _, err := entry.Write(obj.GetBody()); err != nil {
			writeError(w, http.StatusInternalServerError, "download_archive_failed", err.Error())
			return
		}
	}
	if err := zipw.Close(); err != nil {
		writeError(w, http.StatusInternalServerError, "download_archive_failed", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/zip")
	identity := map[string]string{}
	if raw, err := a.getObjectByKey(r, "jobs/"+job.GetId()+"/identity.json"); err == nil {
		_ = json.Unmarshal(raw, &identity)
	}
	w.Header().Set("Content-Disposition", contentDisposition(archiveFileName(identity["brand"], identity["order_month"])))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}

func (a *API) previewMeta(w http.ResponseWriter, r *http.Request) {
	_, job, ok := a.loadJob(w, r)
	if !ok {
		return
	}
	raw, err := a.getObjectByKey(r, preview.MetaKey(job.GetId(), r.PathValue("file_id")))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "preview was not found")
		return
	}
	meta, err := preview.DecodeMeta(raw)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "preview was not found")
		return
	}
	writeJSON(w, http.StatusOK, meta)
}

func (a *API) previewWindow(w http.ResponseWriter, r *http.Request) {
	_, job, ok := a.loadJob(w, r)
	if !ok {
		return
	}
	fileID := r.PathValue("file_id")
	raw, err := a.getObjectByKey(r, preview.MetaKey(job.GetId(), fileID))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "preview was not found")
		return
	}
	meta, err := preview.DecodeMeta(raw)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "preview was not found")
		return
	}
	sheetIndex := queryInt(r, "sheet", 0)
	fromRow := queryInt(r, "from_row", 1)
	toRow := queryInt(r, "to_row", fromRow+99)
	if sheetIndex < 0 || sheetIndex >= len(meta.Sheets) {
		writeError(w, http.StatusNotFound, "not_found", "preview was not found")
		return
	}
	sheet := meta.Sheets[sheetIndex]
	chunkRows := meta.ChunkRows
	fromChunk := preview.ChunkIndex(fromRow, chunkRows)
	toChunk := preview.ChunkIndex(toRow, chunkRows)
	chunks := map[int]preview.Chunk{}
	for index := fromChunk; index <= toChunk; index++ {
		chunkRaw, err := a.getObjectByKey(r, preview.ChunkKey(job.GetId(), fileID, sheet.Index, index))
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", "preview was not found")
			return
		}
		chunk, err := preview.DecodeChunk(chunkRaw)
		if err != nil {
			writeError(w, http.StatusNotFound, "not_found", "preview was not found")
			return
		}
		chunks[index] = chunk
	}
	window := preview.AssembleWindow(sheet, chunkRows, chunks, fromRow, toRow)
	writeJSON(w, http.StatusOK, map[string]any{
		"sheet_index": sheetIndex, "from_row": window.FromRow, "to_row": window.ToRow,
		"rows": window.Rows, "styles": window.Styles,
	})
}

func (a *API) previewFind(w http.ResponseWriter, r *http.Request) {
	_, job, ok := a.loadJob(w, r)
	if !ok {
		return
	}
	raw, err := a.getObjectByKey(r, preview.MetaKey(job.GetId(), r.PathValue("file_id")))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "preview was not found")
		return
	}
	meta, err := preview.DecodeMeta(raw)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "preview was not found")
		return
	}
	sheetIndex := queryInt(r, "sheet", 0)
	if sheetIndex < 0 || sheetIndex >= len(meta.Sheets) {
		writeJSON(w, http.StatusOK, map[string]any{"found": false, "sheet_index": sheetIndex})
		return
	}
	hit := meta.Sheets[sheetIndex].FindArticle(r.URL.Query().Get("q"))
	writeJSON(w, http.StatusOK, map[string]any{"found": hit.Found, "row": hit.Row, "column": hit.Column, "sheet_index": sheetIndex})
}

func (a *API) loadJob(w http.ResponseWriter, r *http.Request) (User, *jobsv1.Job, bool) {
	user, ok := userFrom(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication is required")
		return User{}, nil, false
	}
	resp, err := a.Clients.Jobs.GetJob(a.jobCtx(r, user), &jobsv1.GetJobRequest{Meta: a.meta(user), JobId: r.PathValue("job_id")})
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "job was not found")
		return User{}, nil, false
	}
	return user, resp.GetJob(), true
}

func (a *API) presentJob(ctx context.Context, user User, job *jobsv1.Job) map[string]any {
	created, _ := time.Parse(time.RFC3339, job.GetCreatedAt())
	if created.IsZero() {
		created = time.Now().UTC()
	}
	updated, _ := time.Parse(time.RFC3339, job.GetUpdatedAt())
	if updated.IsZero() {
		updated = created
	}
	identity := map[string]string{}
	if raw, err := a.getObjectByKeyCtx(ctx, "jobs/"+job.GetId()+"/identity.json"); err == nil {
		_ = json.Unmarshal(raw, &identity)
	}
	files, _ := a.Clients.Jobs.ListFiles(grpcutil.WithActorRole(ctx, user.Role), &jobsv1.ListFilesRequest{
		Meta: a.meta(user), JobId: job.GetId(),
	})
	inputs := make([]map[string]any, 0)
	outputs := make([]map[string]any, 0)
	if files != nil {
		for _, file := range files.GetFiles() {
			item := map[string]any{"id": file.GetId(), "role": file.GetKind(), "name": file.GetName(), "content_type": file.GetContentType(), "size_bytes": 0}
			if strings.Contains(file.GetObjectKey(), "/outputs/") || strings.HasPrefix(file.GetId(), "output-") {
				outputs = append(outputs, presentOutputFile(job.GetId(), file))
				continue
			}
			inputs = append(inputs, item)
		}
	}
	out := map[string]any{
		"id": job.GetId(), "type": job.GetType(), "status": job.GetStatus(),
		"brand": identity["brand"], "order_month": identity["order_month"],
		"company_id": job.GetCompanyId(), "created_by": job.GetOwnerUserId(),
		"created_by_login": "",
		"created_at":       created, "updated_at": updated,
		"input_files": inputs, "output_files": outputs, "progress": job.GetProgress(),
	}
	if job.GetErrorMessage() != "" {
		out["error"] = map[string]string{"code": "processing_error", "message": job.GetErrorMessage()}
	}
	if job.GetOwnerUserId() == user.ID {
		out["created_by_login"] = user.Login
	}
	return out
}

func (a *API) withOwnerLogin(ctx context.Context, user User, job *jobsv1.Job, out map[string]any) map[string]any {
	if job.GetOwnerUserId() == user.ID {
		out["created_by_login"] = user.Login
		return out
	}
	out["created_by_login"] = a.ownerLogins(ctx, user, []*jobsv1.Job{job})[job.GetOwnerUserId()]
	return out
}

func (a *API) ownerLogins(ctx context.Context, user User, jobs []*jobsv1.Job) map[string]string {
	logins := map[string]string{user.ID: user.Login}
	if a.Clients.Identity == nil {
		return logins
	}
	seen := map[string]struct{}{}
	for _, job := range jobs {
		companyID := job.GetCompanyId()
		if companyID == "" {
			continue
		}
		if _, ok := seen[companyID]; ok {
			continue
		}
		seen[companyID] = struct{}{}
		resp, err := a.Clients.Identity.ListUsers(ctx, &identityv1.ListUsersRequest{Meta: a.meta(user), CompanyId: companyID})
		if err != nil {
			continue
		}
		for _, item := range resp.GetUsers() {
			logins[item.GetId()] = item.GetLogin()
		}
	}
	return logins
}

func (a *API) getObjectByKey(r *http.Request, key string) ([]byte, error) {
	return a.getObjectByKeyCtx(r.Context(), key)
}

func (a *API) getObjectByKeyCtx(ctx context.Context, key string) ([]byte, error) {
	resp, err := a.Clients.Files.GetObject(ctx, &filesv1.GetObjectRequest{Key: key})
	if err != nil {
		return nil, err
	}
	return resp.GetBody(), nil
}

func queryInt(r *http.Request, name string, fallback int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func archiveFileName(brand, orderMonth string) string {
	safe := strings.TrimSpace(brand)
	if safe == "" {
		safe = "order"
	}
	safe = strings.ReplaceAll(safe, "/", "_")
	month := strings.TrimSpace(orderMonth)
	if month == "" {
		return safe + ".zip"
	}
	return safe + "_" + month + ".zip"
}
