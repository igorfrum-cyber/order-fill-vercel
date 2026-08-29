package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"order-fill/services/api-service/internal/app/usecase"
	"order-fill/services/api-service/internal/domain/job"
)

const maxUploadMemory = 32 << 20

// The driving adapter depends on narrow interfaces rather than concrete use
// cases so handlers stay testable with hand-written fakes.
type (
	jobCreator interface {
		Execute(ctx context.Context, command usecase.CreateJobCommand) (job.Job, error)
	}
	jobFinder interface {
		Execute(ctx context.Context, jobID string) (job.Job, error)
	}
	reportFinder interface {
		Execute(ctx context.Context, jobID string) (job.Report, error)
	}
	fileLister interface {
		Execute(ctx context.Context, jobID string) ([]job.OutputFile, error)
	}
	fileDownloader interface {
		Execute(ctx context.Context, jobID string, fileID string) (usecase.Download, error)
	}
	archiveDownloader interface {
		Execute(ctx context.Context, jobID string) (usecase.Download, error)
	}
	editSubmitter interface {
		Execute(ctx context.Context, jobID string, edits []job.ManualEdit) (job.Job, error)
	}
)

type jobHandler struct {
	creator    jobCreator
	finder     jobFinder
	reports    reportFinder
	files      fileLister
	downloads  fileDownloader
	archive    archiveDownloader
	editor     editSubmitter
	maxUploads int64
}

func (h jobHandler) createOrderFill(w http.ResponseWriter, r *http.Request) {
	h.create(w, r, job.TypeOrderFill)
}

func (h jobHandler) createNorthMerge(w http.ResponseWriter, r *http.Request) {
	h.create(w, r, job.TypeNorthMerge)
}

func (h jobHandler) create(w http.ResponseWriter, r *http.Request, jobType job.Type) {
	if h.creator == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "job creation is not configured")
		return
	}
	command, err := h.parseCreateRequest(r, jobType)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	entity, err := h.creator.Execute(r.Context(), command)
	if err != nil {
		writeDomainError(w, "create_job_failed", err)
		return
	}
	writeJSON(w, http.StatusAccepted, presentJob(entity))
}

func (h jobHandler) getJob(w http.ResponseWriter, r *http.Request) {
	if h.finder == nil {
		writeError(w, http.StatusNotFound, "not_found", "job was not found")
		return
	}
	entity, err := h.finder.Execute(r.Context(), r.PathValue("job_id"))
	if err != nil {
		writeDomainError(w, "read_job_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, presentJob(entity))
}

func (h jobHandler) getReport(w http.ResponseWriter, r *http.Request) {
	if h.reports == nil {
		writeError(w, http.StatusNotFound, "not_found", "report was not found")
		return
	}
	report, err := h.reports.Execute(r.Context(), r.PathValue("job_id"))
	if err != nil {
		writeDomainError(w, "read_report_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, presentReport(report))
}

func (h jobHandler) submitEdits(w http.ResponseWriter, r *http.Request) {
	if h.editor == nil {
		writeError(w, http.StatusNotFound, "not_found", "job was not found")
		return
	}
	var payload struct {
		Edits []struct {
			Key     string `json:"key"`
			Value   string `json:"value"`
			Comment string `json:"comment"`
		} `json:"edits"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	edits := make([]job.ManualEdit, 0, len(payload.Edits))
	for _, edit := range payload.Edits {
		edits = append(edits, job.ManualEdit{Key: edit.Key, Value: edit.Value, Comment: edit.Comment})
	}
	entity, err := h.editor.Execute(r.Context(), r.PathValue("job_id"), edits)
	if err != nil {
		writeDomainError(w, "submit_edits_failed", err)
		return
	}
	writeJSON(w, http.StatusAccepted, presentJob(entity))
}

func (h jobHandler) listFiles(w http.ResponseWriter, r *http.Request) {
	if h.files == nil {
		writeError(w, http.StatusNotFound, "not_found", "job was not found")
		return
	}
	jobID := r.PathValue("job_id")
	files, err := h.files.Execute(r.Context(), jobID)
	if err != nil {
		writeDomainError(w, "read_files_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string][]outputFileResponse{"files": presentOutputFiles(jobID, files)})
}

func (h jobHandler) downloadFile(w http.ResponseWriter, r *http.Request) {
	if h.downloads == nil {
		writeError(w, http.StatusNotFound, "not_found", "file was not found")
		return
	}
	download, err := h.downloads.Execute(r.Context(), r.PathValue("job_id"), r.PathValue("file_id"))
	if err != nil {
		writeDomainError(w, "download_failed", err)
		return
	}
	w.Header().Set("Content-Type", download.ContentType)
	w.Header().Set("Content-Disposition", contentDisposition(download.Name))
	w.Header().Set("Content-Length", fmt.Sprint(len(download.Content)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(download.Content)
}

func (h jobHandler) downloadArchive(w http.ResponseWriter, r *http.Request) {
	if h.archive == nil {
		writeError(w, http.StatusNotFound, "not_found", "file was not found")
		return
	}
	download, err := h.archive.Execute(r.Context(), r.PathValue("job_id"))
	if err != nil {
		writeDomainError(w, "download_archive_failed", err)
		return
	}
	w.Header().Set("Content-Type", download.ContentType)
	w.Header().Set("Content-Disposition", contentDisposition(download.Name))
	w.Header().Set("Content-Length", fmt.Sprint(len(download.Content)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(download.Content)
}

// contentDisposition keeps Cyrillic file names readable in every browser by
// sending both the ASCII fallback and the RFC 5987 encoded name.
func contentDisposition(name string) string {
	ascii := strings.Map(func(char rune) rune {
		if char < 32 || char > 126 || char == '"' || char == '\\' {
			return '_'
		}
		return char
	}, name)
	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, ascii, url.PathEscape(name))
}

func (h jobHandler) parseCreateRequest(r *http.Request, jobType job.Type) (usecase.CreateJobCommand, error) {
	limit := h.maxUploads
	if limit <= 0 {
		limit = maxUploadMemory
	}
	if err := r.ParseMultipartForm(limit); err != nil {
		return usecase.CreateJobCommand{}, err
	}

	uploads := make([]job.Upload, 0)
	if jobType == job.TypeOrderFill {
		sources, err := multipartUploads(r.MultipartForm, "source_file", job.RoleSource, true)
		if err != nil {
			return usecase.CreateJobCommand{}, err
		}
		uploads = append(uploads, sources...)
	}
	if jobType == job.TypeNorthMerge {
		sources, err := multipartUploads(r.MultipartForm, "tyumen_source_file", job.RoleSource, false)
		if err != nil {
			return usecase.CreateJobCommand{}, err
		}
		uploads = append(uploads, sources...)
	}
	blanks, err := multipartUploads(r.MultipartForm, "blank_files", job.RoleBlank, true)
	if err != nil {
		return usecase.CreateJobCommand{}, err
	}
	uploads = append(uploads, blanks...)

	return usecase.CreateJobCommand{
		Type:       jobType,
		Brand:      r.FormValue("brand"),
		OrderMonth: r.FormValue("order_month"),
		Uploads:    uploads,
	}, nil
}

func multipartUploads(form *multipart.Form, field string, role job.Role, required bool) ([]job.Upload, error) {
	if form == nil {
		return nil, errors.New("multipart form is required")
	}
	headers := form.File[field]
	if len(headers) == 0 {
		if required {
			return nil, errors.New(field + " is required")
		}
		return nil, nil
	}

	uploads := make([]job.Upload, 0, len(headers))
	for _, header := range headers {
		file, err := header.Open()
		if err != nil {
			return nil, err
		}
		content, readErr := io.ReadAll(file)
		closeErr := file.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		uploads = append(uploads, job.Upload{
			Role:        role,
			Name:        header.Filename,
			ContentType: header.Header.Get("Content-Type"),
			Content:     content,
		})
	}
	return uploads, nil
}

func writeDomainError(w http.ResponseWriter, code string, err error) {
	switch {
	case errors.Is(err, job.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, job.ErrInvalid):
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, code, err.Error())
	}
}
