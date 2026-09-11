package grpcapi

import (
	"context"

	"google.golang.org/grpc"

	"order-fill/backend/pkg/grpcutil"
	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	filesv1 "order-fill/backend/proto/gen/go/orderfill/files/v1"
	"order-fill/backend/services/file-service/internal/clients/identity"
	"order-fill/backend/services/file-service/internal/domain"
	"order-fill/backend/services/file-service/internal/service/files"
)

const rolePlatformAdmin = "platform_admin"

type Server struct {
	filesv1.UnimplementedFileServiceServer
	svc         *files.Service
	actors      ActorLookup
	workerToken string
}

type ActorLookup interface {
	Actor(ctx context.Context, userID string) (identity.Actor, error)
}

func NewServer(svc *files.Service, actors ActorLookup, workerToken string) *Server {
	return &Server{svc: svc, actors: actors, workerToken: workerToken}
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

func (s *Server) openACL() bool {
	return s.actors == nil && s.workerToken == ""
}

func (s *Server) principal(ctx context.Context, meta *commonv1.RequestMeta) (bool, identity.Actor, error) {
	if grpcutil.WorkerAuthorized(ctx, s.workerToken) {
		return true, identity.Actor{}, nil
	}
	if s.openACL() {
		return false, identity.Actor{}, nil
	}
	if s.actors == nil {
		return false, identity.Actor{}, domain.ErrUnauthorized
	}
	userID := ""
	if meta != nil {
		userID = meta.GetActorUserId()
	}
	actor, err := s.actors.Actor(ctx, userID)
	return false, actor, err
}

func canRead(worker bool, actor identity.Actor, obj domain.Object) bool {
	if worker {
		return true
	}
	if domain.PublicLogoKey(obj.Key) {
		return true
	}
	if actor.Role == rolePlatformAdmin {
		return true
	}
	return obj.CompanyID != "" && obj.CompanyID == actor.CompanyID
}

func stampCompany(worker bool, actor identity.Actor, meta *commonv1.RequestMeta, key string) (string, error) {
	if worker {
		if meta == nil || meta.GetCompanyId() == "" {
			return "", domain.ErrInvalid
		}
		return meta.GetCompanyId(), nil
	}
	if domain.PublicLogoKey(key) {
		logoCompany := domain.LogoCompanyID(key)
		if actor.Role == rolePlatformAdmin || actor.CompanyID == logoCompany {
			return logoCompany, nil
		}
		return "", domain.ErrUnauthorized
	}
	if actor.Role == rolePlatformAdmin && meta != nil && meta.GetCompanyId() != "" {
		return meta.GetCompanyId(), nil
	}
	if actor.CompanyID == "" {
		return "", domain.ErrUnauthorized
	}
	return actor.CompanyID, nil
}

func (s *Server) PutObject(ctx context.Context, req *filesv1.PutObjectRequest) (*filesv1.PutObjectResponse, error) {
	worker, actor, err := s.principal(ctx, req.GetMeta())
	if err != nil {
		return nil, err
	}
	companyID := ""
	if !s.openACL() {
		companyID, err = stampCompany(worker, actor, req.GetMeta(), req.GetKey())
		if err != nil {
			return nil, err
		}
	} else if req.GetMeta() != nil {
		companyID = req.GetMeta().GetCompanyId()
	}
	obj, err := s.svc.Put(ctx, req.GetKey(), req.GetName(), req.GetContentType(), req.GetBody(), companyID)
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
	if s.openACL() || domain.PublicLogoKey(obj.Key) {
		return &filesv1.GetObjectResponse{Object: protoObject(obj), Body: obj.Body}, nil
	}
	worker, actor, err := s.principal(ctx, req.GetMeta())
	if err != nil {
		return nil, domain.ErrNotFound
	}
	if !canRead(worker, actor, obj) {
		return nil, domain.ErrNotFound
	}
	return &filesv1.GetObjectResponse{Object: protoObject(obj), Body: obj.Body}, nil
}

func (s *Server) CreateUpload(ctx context.Context, req *filesv1.CreateUploadRequest) (*filesv1.CreateUploadResponse, error) {
	if _, _, err := s.principal(ctx, req.GetMeta()); err != nil {
		return nil, err
	}
	up, err := s.svc.CreateUpload(ctx, req.GetName(), req.GetContentType())
	if err != nil {
		return nil, err
	}
	return &filesv1.CreateUploadResponse{UploadId: up.ID}, nil
}

func (s *Server) FinalizeUpload(ctx context.Context, req *filesv1.FinalizeUploadRequest) (*filesv1.FinalizeUploadResponse, error) {
	worker, actor, err := s.principal(ctx, req.GetMeta())
	if err != nil {
		return nil, err
	}
	companyID := ""
	if !s.openACL() {
		companyID, err = stampCompany(worker, actor, req.GetMeta(), "")
		if err != nil {
			return nil, err
		}
	}
	obj, err := s.svc.FinalizeUpload(ctx, req.GetUploadId(), req.GetBody(), companyID)
	if err != nil {
		return nil, err
	}
	return &filesv1.FinalizeUploadResponse{Object: protoObject(obj)}, nil
}

func (s *Server) CreateArchive(ctx context.Context, req *filesv1.CreateArchiveRequest) (*filesv1.CreateArchiveResponse, error) {
	worker, actor, err := s.principal(ctx, req.GetMeta())
	if err != nil {
		return nil, err
	}
	companyID := ""
	for _, ref := range req.GetObjectIds() {
		obj, err := s.svc.Get(ctx, ref, "")
		if err != nil {
			obj, err = s.svc.Get(ctx, "", ref)
		}
		if err != nil {
			return nil, err
		}
		if !s.openACL() && !canRead(worker, actor, obj) {
			return nil, domain.ErrNotFound
		}
		if companyID == "" {
			companyID = obj.CompanyID
		}
	}
	if !s.openACL() && worker {
		if req.GetMeta() != nil && req.GetMeta().GetCompanyId() != "" {
			companyID = req.GetMeta().GetCompanyId()
		}
		if companyID == "" {
			return nil, domain.ErrInvalid
		}
	}
	if !s.openACL() && !worker && companyID == "" {
		stamped, err := stampCompany(false, actor, req.GetMeta(), "")
		if err != nil {
			return nil, err
		}
		companyID = stamped
	}
	obj, err := s.svc.Archive(ctx, req.GetObjectIds(), req.GetName(), companyID)
	if err != nil {
		return nil, err
	}
	return &filesv1.CreateArchiveResponse{Object: protoObject(obj)}, nil
}
