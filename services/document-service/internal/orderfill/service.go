package orderfill

import "context"

type InputFile struct {
	Name    string
	Content []byte
}

type Report struct {
	Rows []ReportRow
}

type ReportRow struct {
	Key      string
	Status   string
	Editable bool
}

type Processor interface {
	Process(ctx context.Context, files []InputFile) (Report, error)
}

type ProcessorFunc func(ctx context.Context, files []InputFile) (Report, error)

func (f ProcessorFunc) Process(ctx context.Context, files []InputFile) (Report, error) {
	return f(ctx, files)
}
