package usecase

import (
	"context"
	"fmt"

	"order-fill/services/api-service/internal/app/port"
	"order-fill/services/api-service/internal/domain/job"
	"order-fill/services/api-service/internal/domain/preview"
)

// PreviewReader serves workbook grid windows from sidecar objects. It never
// loads a whole 100k-row sheet: meta is one gzip blob, a window is one or two
// chunks.
type PreviewReader struct {
	repository JobReader
	storage    port.ObjectStore
}

// PreviewWindowQuery is the browser viewport translated into sheet coordinates.
type PreviewWindowQuery struct {
	JobID      string
	FileID     string
	SheetIndex int
	FromRow    int
	ToRow      int
}

func NewPreviewReader(repository JobReader, storage port.ObjectStore) *PreviewReader {
	return &PreviewReader{repository: repository, storage: storage}
}

func (u *PreviewReader) Meta(ctx context.Context, jobID string, fileID string) (preview.Meta, error) {
	if _, err := u.outputFile(ctx, jobID, fileID); err != nil {
		return preview.Meta{}, err
	}
	object, err := u.storage.Get(ctx, preview.MetaKey(jobID, fileID))
	if err != nil {
		return preview.Meta{}, fmt.Errorf("read preview meta: %w", err)
	}
	meta, err := preview.DecodeMeta(object.Content)
	if err != nil {
		return preview.Meta{}, fmt.Errorf("decode preview meta: %w", err)
	}
	return meta, nil
}

func (u *PreviewReader) Window(ctx context.Context, query PreviewWindowQuery) (preview.Window, error) {
	meta, err := u.Meta(ctx, query.JobID, query.FileID)
	if err != nil {
		return preview.Window{}, err
	}
	sheet, err := sheetAt(meta, query.SheetIndex)
	if err != nil {
		return preview.Window{}, err
	}
	fromRow, toRow, err := clampWindow(query.FromRow, query.ToRow, sheet.MaxRow)
	if err != nil {
		return preview.Window{}, err
	}
	chunkRows := meta.ChunkRows
	fromChunk := preview.ChunkIndex(fromRow, chunkRows)
	toChunk := preview.ChunkIndex(toRow, chunkRows)
	chunks := make(map[int]preview.Chunk, toChunk-fromChunk+1)
	for index := fromChunk; index <= toChunk; index++ {
		object, err := u.storage.Get(ctx, preview.ChunkKey(query.JobID, query.FileID, sheet.Index, index))
		if err != nil {
			return preview.Window{}, fmt.Errorf("read preview chunk: %w", err)
		}
		chunk, err := preview.DecodeChunk(object.Content)
		if err != nil {
			return preview.Window{}, fmt.Errorf("decode preview chunk: %w", err)
		}
		chunks[index] = chunk
	}
	return preview.AssembleWindow(sheet, chunkRows, chunks, fromRow, toRow), nil
}

func (u *PreviewReader) Find(ctx context.Context, jobID string, fileID string, sheetIndex int, query string) (preview.Hit, error) {
	meta, err := u.Meta(ctx, jobID, fileID)
	if err != nil {
		return preview.Hit{}, err
	}
	sheet, err := sheetAt(meta, sheetIndex)
	if err != nil {
		return preview.Hit{}, err
	}
	return sheet.FindArticle(query), nil
}

func (u *PreviewReader) outputFile(ctx context.Context, jobID string, fileID string) (job.OutputFile, error) {
	entity, err := u.repository.Get(ctx, jobID)
	if err != nil {
		return job.OutputFile{}, err
	}
	for _, file := range entity.OutputFiles {
		if file.ID == fileID {
			return file, nil
		}
	}
	return job.OutputFile{}, fmt.Errorf("%w: file %q", job.ErrNotFound, fileID)
}

func sheetAt(meta preview.Meta, sheetIndex int) (preview.SheetMeta, error) {
	if sheetIndex < 0 || sheetIndex >= len(meta.Sheets) {
		return preview.SheetMeta{}, fmt.Errorf("%w: preview sheet %d", job.ErrInvalid, sheetIndex)
	}
	return meta.Sheets[sheetIndex], nil
}

func clampWindow(fromRow int, toRow int, maxRow int) (int, int, error) {
	if fromRow < 1 {
		fromRow = 1
	}
	if toRow < 1 {
		toRow = fromRow
	}
	if toRow < fromRow {
		return 0, 0, fmt.Errorf("%w: to_row must be >= from_row", job.ErrInvalid)
	}
	if toRow-fromRow+1 > preview.MaxWindowRows {
		return 0, 0, fmt.Errorf("%w: preview window cannot exceed %d rows", job.ErrInvalid, preview.MaxWindowRows)
	}
	if maxRow > 0 && fromRow > maxRow {
		fromRow = maxRow
		toRow = maxRow
	}
	return fromRow, toRow, nil
}
