package grpcapi

import (
	"context"

	"google.golang.org/grpc"

	"order-fill/backend/pkg/grpcutil"
	filesv1 "order-fill/backend/proto/gen/go/orderfill/files/v1"
	"order-fill/backend/services/file-service/internal/domain"
	"order-fill/backend/services/file-service/internal/service/files"
)

type Server struct {
	filesv1.UnimplementedFileServiceServer
	svc *files.Service
}

func NewServer(svc *files.Service) *Server {
	return &Server{svc: svc}
}

func New(handler filesv1.FileServiceServer) *grpc.Server {
	s := grpcutil.NewServer()
	if handler != nil {
		filesv1.RegisterFileServiceServer(s, handler)
	}
	return s
}

func protoObject(obj domain.Object) *filesv1.ObjectMeta {
	return &filesv1.ObjectMeta{
		Id: obj.ID, Key: obj.Key, Name: obj.Name, ContentType: obj.ContentType, Size: obj.Size,
	}
}

func (s *Server) PutObject(ctx context.Context, req *filesv1.PutObjectRequest) (*filesv1.PutObjectResponse, error) {
	obj, err := s.svc.Put(ctx, req.GetKey(), req.GetName(), req.GetContentType(), req.GetBody())
	if err != nil {
		return nil, err
	}
	return &filesv1.PutObjectResponse{Object: protoObject(obj)}, nil
}

func (s *Server) GetObject(ctx context.Context, req *filesv1.GetObjectRequest) (*filesv1.GetObjectResponse, error) {
	obj, err := s.svc.Get(ctx, req.GetId(), req.GetKey())
	if err != nil {
		return nil, err
	}
	return &filesv1.GetObjectResponse{Object: protoObject(obj), Body: obj.Body}, nil
}

func (s *Server) CreateUpload(ctx context.Context, req *filesv1.CreateUploadRequest) (*filesv1.CreateUploadResponse, error) {
	up, err := s.svc.CreateUpload(ctx, req.GetName(), req.GetContentType())
	if err != nil {
		return nil, err
	}
	return &filesv1.CreateUploadResponse{UploadId: up.ID}, nil
}

func (s *Server) FinalizeUpload(ctx context.Context, req *filesv1.FinalizeUploadRequest) (*filesv1.FinalizeUploadResponse, error) {
	obj, err := s.svc.FinalizeUpload(ctx, req.GetUploadId(), req.GetBody())
	if err != nil {
		return nil, err
	}
	return &filesv1.FinalizeUploadResponse{Object: protoObject(obj)}, nil
}

func (s *Server) CreateArchive(ctx context.Context, req *filesv1.CreateArchiveRequest) (*filesv1.CreateArchiveResponse, error) {
	obj, err := s.svc.Archive(ctx, req.GetObjectIds(), req.GetName())
	if err != nil {
		return nil, err
	}
	return &filesv1.CreateArchiveResponse{Object: protoObject(obj)}, nil
}
