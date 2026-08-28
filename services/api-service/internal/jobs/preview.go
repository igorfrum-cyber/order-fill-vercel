package jobs

import (
	"context"
	"fmt"
	"net/url"
)

type PreviewProcessor interface {
	ProcessPreview(ctx context.Context, job Job) (Report, []OutputFile, error)
}

type StaticPreviewProcessor struct{}

func (p StaticPreviewProcessor) ProcessPreview(_ context.Context, job Job) (Report, []OutputFile, error) {
	report := Report{
		JobID: job.ID,
		Rows: []ReportRow{{
			Key:      "preview:" + job.ID,
			Status:   "matched",
			Editable: true,
		}},
	}
	files := []OutputFile{{
		ID:          "preview-output",
		Label:       "Скачать preview результат",
		Name:        fmt.Sprintf("%s-preview.txt", job.ID),
		ContentType: "text/plain; charset=utf-8",
		SizeBytes:   40,
		DownloadURL: "data:text/plain;charset=utf-8," + url.QueryEscape("Preview output. Real Excel generation runs in document-service."),
	}}
	return report, files, nil
}
