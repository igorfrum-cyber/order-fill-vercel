package files

import (
	"context"
	"io"

	"order-fill/backend/services/file-service/internal/domain"
)

func (s *Service) Get(ctx context.Context, id, key string) (domain.Object, error) {
	var obj domain.Object
	var err error
	if id != "" {
		obj, err = s.meta.GetByID(id)
	} else {
		obj, err = s.meta.GetByKey(key)
	}
	if err != nil {
		return domain.Object{}, domain.ErrNotFound
	}
	rc, contentType, err := s.blobs.Get(ctx, obj.Key)
	if err != nil {
		return domain.Object{}, domain.ErrNotFound
	}
	defer rc.Close()
	body, err := io.ReadAll(rc)
	if err != nil {
		return domain.Object{}, err
	}
	obj.Body = body
	if obj.ContentType == "" {
		obj.ContentType = contentType
	}
	obj.Size = int64(len(body))
	return obj, nil
}
