package httpapi

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"

	"order-fill/services/api-service/internal/jobs"
)

const maxUploadMemory = 32 << 20

type JobCreator interface {
	CreateJob(ctx context.Context, command jobs.CreateJobCommand) (jobs.Job, error)
}

type JobReader interface {
	Find(ctx context.Context, id string) (jobs.Job, error)
}

type jobHandler struct {
	creator JobCreator
	reader  JobReader
}

func (h jobHandler) createOrderFill(w http.ResponseWriter, r *http.Request) {
	command, err := h.parseCreateJobRequest(r, jobs.JobTypeOrderFill)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	job, err := h.creator.CreateJob(r.Context(), command)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, jobs.ErrInvalidJob) {
			status = http.StatusBadRequest
		}
		writeError(w, status, "create_job_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}

func (h jobHandler) createNorthMerge(w http.ResponseWriter, r *http.Request) {
	command, err := h.parseCreateJobRequest(r, jobs.JobTypeNorthMerge)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	job, err := h.creator.CreateJob(r.Context(), command)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, jobs.ErrInvalidJob) {
			status = http.StatusBadRequest
		}
		writeError(w, status, "create_job_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}

func (h jobHandler) getJob(w http.ResponseWriter, r *http.Request) {
	if h.reader == nil {
		writeError(w, http.StatusNotFound, "not_found", "job was not found")
		return
	}
	job, err := h.reader.Find(r.Context(), r.PathValue("job_id"))
	if err != nil {
		status := http.StatusInternalServerError
		code := "read_job_failed"
		if errors.Is(err, jobs.ErrJobNotFound) {
			status = http.StatusNotFound
			code = "not_found"
		}
		writeError(w, status, code, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (h jobHandler) parseCreateJobRequest(r *http.Request, jobType jobs.JobType) (jobs.CreateJobCommand, error) {
	if h.creator == nil {
		return jobs.CreateJobCommand{}, errors.New("job creator is not configured")
	}
	if err := r.ParseMultipartForm(maxUploadMemory); err != nil {
		return jobs.CreateJobCommand{}, err
	}

	files := make([]jobs.UploadFile, 0)
	if jobType == jobs.JobTypeOrderFill {
		sourceFiles, err := multipartFiles(r.MultipartForm, "source_file", true)
		if err != nil {
			return jobs.CreateJobCommand{}, err
		}
		files = append(files, sourceFiles...)
	}
	if jobType == jobs.JobTypeNorthMerge {
		sourceFiles, err := multipartFiles(r.MultipartForm, "tyumen_source_file", false)
		if err != nil {
			return jobs.CreateJobCommand{}, err
		}
		files = append(files, sourceFiles...)
	}
	blankFiles, err := multipartFiles(r.MultipartForm, "blank_files", true)
	if err != nil {
		return jobs.CreateJobCommand{}, err
	}
	files = append(files, blankFiles...)

	return jobs.CreateJobCommand{
		Type:       jobType,
		Brand:      r.FormValue("brand"),
		OrderMonth: r.FormValue("order_month"),
		Files:      files,
	}, nil
}

func multipartFiles(form *multipart.Form, field string, required bool) ([]jobs.UploadFile, error) {
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

	files := make([]jobs.UploadFile, 0, len(headers))
	for _, header := range headers {
		file, err := header.Open()
		if err != nil {
			return nil, err
		}
		content, err := io.ReadAll(file)
		closeErr := file.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
		files = append(files, jobs.UploadFile{
			Name:        header.Filename,
			ContentType: header.Header.Get("Content-Type"),
			SizeBytes:   int64(len(content)),
			Reader:      bytes.NewReader(content),
		})
	}
	return files, nil
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, apiError{Code: code, Message: message})
}

type apiError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}
