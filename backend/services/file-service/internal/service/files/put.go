package files

import (
	"bytes"
	"context"
	"fmt"

	"order-fill/backend/services/file-service/internal/domain"
)

func (s *Service) Put(ctx context.Context, key, name, contentType string, body []byte) (domain.Object, error) {
	name = domain.SafeFileName(name)
	if name == "file" && key == "" {
		return domain.Object{}, fmt.Errorf("%w: name is required", domain.ErrInvalid)
	}
	if key == "" {
		id, err := newID()
		if err != nil {
			return domain.Object{}, err
		}
		key = "objects/" + id + "/" + name
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if err := s.blobs.Put(ctx, key, bytes.NewReader(body), int64(len(body)), contentType); err != nil {
		return domain.Object{}, err
	}
	id, err := newID()
	if err != nil {
		return domain.Object{}, err
	}
	obj := domain.Object{ID: id, Key: key, Name: name, ContentType: contentType, Size: int64(len(body))}
	if err := s.meta.SaveObject(obj); err != nil {
		return domain.Object{}, fmt.Errorf("save object meta: %w", err)
	}
	return obj, nil
}

func (s *Service) CreateUpload(_ context.Context, name, contentType string) (domain.Upload, error) {
	id, err := newID()
	if err != nil {
		return domain.Upload{}, err
	}
	up := domain.Upload{ID: id, Name: domain.SafeFileName(name), ContentType: contentType}
	if err := s.meta.SaveUpload(up); err != nil {
		return domain.Upload{}, fmt.Errorf("save upload meta: %w", err)
	}
	return up, nil
}

func (s *Service) FinalizeUpload(ctx context.Context, uploadID string, body []byte) (domain.Object, error) {
	up, err := s.meta.GetUpload(uploadID)
	if err != nil {
		return domain.Object{}, err
	}
	if up.ObjectID != "" {
		return s.meta.GetByID(up.ObjectID)
	}
	obj, err := s.Put(ctx, "", up.Name, up.ContentType, body)
	if err != nil {
		return domain.Object{}, err
	}
	up.ObjectID = obj.ID
	if err := s.meta.SaveUpload(up); err != nil {
		return domain.Object{}, fmt.Errorf("save upload meta: %w", err)
	}
	return obj, nil
}
