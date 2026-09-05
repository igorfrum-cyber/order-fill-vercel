package files

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"

	"order-fill/backend/services/file-service/internal/domain"
)

func (s *Service) Archive(ctx context.Context, objectIDs []string, name string) (domain.Object, error) {
	if len(objectIDs) == 0 {
		return domain.Object{}, fmt.Errorf("%w: archive needs files", domain.ErrInvalid)
	}
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, id := range objectIDs {
		obj, err := s.Get(ctx, id, "")
		if err != nil {
			_ = zw.Close()
			return domain.Object{}, err
		}
		w, err := zw.Create(domain.SafeFileName(obj.Name))
		if err != nil {
			_ = zw.Close()
			return domain.Object{}, err
		}
		if _, err := w.Write(obj.Body); err != nil {
			_ = zw.Close()
			return domain.Object{}, err
		}
	}
	if err := zw.Close(); err != nil {
		return domain.Object{}, err
	}
	name = domain.SafeFileName(name)
	if name == "file" {
		name = "archive.zip"
	}
	return s.Put(ctx, "", name, "application/zip", buf.Bytes())
}
