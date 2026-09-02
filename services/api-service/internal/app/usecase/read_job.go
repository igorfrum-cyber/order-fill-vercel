package usecase

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"path"
	"strings"

	"order-fill/services/api-service/internal/app/port"
	"order-fill/services/api-service/internal/domain/job"
)

// JobReader is the narrow slice of the repository the read use cases need.
type JobReader interface {
	Get(ctx context.Context, id string) (job.Job, error)
}

// GetJob returns a single job with its current status.
type GetJob struct {
	repository JobReader
}

func NewGetJob(repository JobReader) *GetJob {
	return &GetJob{repository: repository}
}

func (u *GetJob) Execute(ctx context.Context, jobID string) (job.Job, error) {
	return u.repository.Get(ctx, jobID)
}

// GetReport returns the reviewable report of a job.
type GetReport struct {
	reports port.ReportReader
}

func NewGetReport(reports port.ReportReader) *GetReport {
	return &GetReport{reports: reports}
}

func (u *GetReport) Execute(ctx context.Context, jobID string) (job.Report, error) {
	return u.reports.Report(ctx, jobID)
}

// ListFiles returns the generated files of a job.
type ListFiles struct {
	repository JobReader
}

func NewListFiles(repository JobReader) *ListFiles {
	return &ListFiles{repository: repository}
}

func (u *ListFiles) Execute(ctx context.Context, jobID string) ([]job.OutputFile, error) {
	entity, err := u.repository.Get(ctx, jobID)
	if err != nil {
		return nil, err
	}
	return append([]job.OutputFile(nil), entity.OutputFiles...), nil
}

// DownloadFile streams a generated file back to the browser.
type DownloadFile struct {
	repository JobReader
	storage    port.ObjectStore
}

// Download is a generated file ready to be written to an HTTP response.
type Download struct {
	Name        string
	ContentType string
	Content     []byte
}

func NewDownloadFile(repository JobReader, storage port.ObjectStore) *DownloadFile {
	return &DownloadFile{repository: repository, storage: storage}
}

func (u *DownloadFile) Execute(ctx context.Context, jobID string, fileID string) (Download, error) {
	entity, err := u.repository.Get(ctx, jobID)
	if err != nil {
		return Download{}, err
	}
	for _, file := range entity.OutputFiles {
		if file.ID != fileID {
			continue
		}
		object, err := u.storage.Get(ctx, file.StorageKey)
		if err != nil {
			return Download{}, fmt.Errorf("read output file: %w", err)
		}
		contentType := file.ContentType
		if contentType == "" {
			contentType = object.ContentType
		}
		return Download{Name: file.Name, ContentType: contentType, Content: object.Content}, nil
	}
	return Download{}, fmt.Errorf("%w: file %q", job.ErrNotFound, fileID)
}

// DownloadArchive packs every generated workbook into one zip for a single click.
type DownloadArchive struct {
	repository JobReader
	storage    port.ObjectStore
}

func NewDownloadArchive(repository JobReader, storage port.ObjectStore) *DownloadArchive {
	return &DownloadArchive{repository: repository, storage: storage}
}

func (u *DownloadArchive) Execute(ctx context.Context, jobID string) (Download, error) {
	entity, err := u.repository.Get(ctx, jobID)
	if err != nil {
		return Download{}, err
	}
	if len(entity.OutputFiles) == 0 {
		return Download{}, fmt.Errorf("%w: generated files", job.ErrNotFound)
	}

	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	used := map[string]int{}
	for _, file := range entity.OutputFiles {
		object, err := u.storage.Get(ctx, file.StorageKey)
		if err != nil {
			return Download{}, fmt.Errorf("read output file: %w", err)
		}
		entryName := uniqueZipName(job.SafeFileName(file.Name), used)
		entry, err := writer.Create(entryName)
		if err != nil {
			return Download{}, fmt.Errorf("create archive entry: %w", err)
		}
		if _, err := entry.Write(object.Content); err != nil {
			return Download{}, fmt.Errorf("write archive entry: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return Download{}, fmt.Errorf("close archive: %w", err)
	}
	return Download{
		Name:        job.ArchiveFileName(entity.Brand, entity.OrderMonth),
		ContentType: "application/zip",
		Content:     buffer.Bytes(),
	}, nil
}

func uniqueZipName(name string, used map[string]int) string {
	if name == "" || name == "file" {
		name = "file.xlsx"
	}
	count := used[name]
	used[name] = count + 1
	if count == 0 {
		return name
	}
	ext := path.Ext(name)
	base := strings.TrimSuffix(name, ext)
	return fmt.Sprintf("%s-%d%s", base, count+1, ext)
}
