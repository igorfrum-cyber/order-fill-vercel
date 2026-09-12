package grpcapi

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"order-fill/backend/pkg/grpcutil"
	documentsv1 "order-fill/backend/proto/gen/go/orderfill/documents/v1"
	filesv1 "order-fill/backend/proto/gen/go/orderfill/files/v1"
	"order-fill/backend/services/document-service/internal/clients/brand"
	"order-fill/backend/services/document-service/internal/domain/orderfill"
	"order-fill/backend/services/document-service/internal/domain/preview"
	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
)

type Server struct {
	documentsv1.UnimplementedDocumentServiceServer
	files  filesv1.FileServiceClient
	codec  spreadsheet.Codec
	brands brand.Client
	token  string
}

func NewServer(files filesv1.FileServiceClient, codec spreadsheet.Codec, brands brand.Client, token string) *Server {
	return &Server{files: files, codec: codec, brands: brands, token: token}
}

func New(handler documentsv1.DocumentServiceServer) *grpc.Server {
	s := grpcutil.NewServer()
	if handler != nil {
		documentsv1.RegisterDocumentServiceServer(s, handler)
	}
	return s
}

func (s *Server) fileCtx(ctx context.Context) context.Context {
	return grpcutil.WithWorkerToken(ctx, s.token)
}

func (s *Server) AnalyzeInputs(ctx context.Context, req *documentsv1.AnalyzeInputsRequest) (*documentsv1.AnalyzeInputsResponse, error) {
	if s.files == nil || s.codec == nil || s.brands == nil {
		return nil, status.Error(codes.Unavailable, "document api is not configured")
	}
	var group, blankName string
	var blankNames []string
	var blankBook spreadsheet.Workbook
	for _, id := range req.GetInputFileIds() {
		obj, err := s.files.GetObject(s.fileCtx(ctx), &filesv1.GetObjectRequest{Meta: req.GetMeta(), Id: id})
		if err != nil {
			return nil, err
		}
		book, err := s.codec.Load(obj.GetBody())
		if err != nil {
			continue
		}
		if g, err := orderfill.NomenclatureGroup(book); err == nil {
			group = g
			continue
		}
		name := obj.GetObject().GetName()
		blankNames = append(blankNames, name)
		if blankName == "" {
			blankName = name
		}
		blankBook = book
	}
	if group == "" {
		return nil, status.Error(codes.InvalidArgument, "source workbook was not found")
	}
	detected, _, err := s.brands.Detect(ctx, group, blankName)
	if err != nil {
		return nil, err
	}
	if detected == "" {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("не узнали бренд «%s». Проверьте отбор номенклатуры в выгрузке 1С", group))
	}
	plans, err := orderfill.PlanBlanks(detected, blankNames)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	blankLabel := ""
	if len(plans) > 0 {
		blankLabel = plans[0].Label
		if detected == "christina" && blankBook != nil {
			blankLabel = orderfill.LabelChristinaBlank(blankBook, blankLabel)
		}
	}
	return &documentsv1.AnalyzeInputsResponse{Brand: detected, BlankLabel: blankLabel}, nil
}

func (s *Server) BuildPreview(ctx context.Context, req *documentsv1.BuildPreviewRequest) (*documentsv1.BuildPreviewResponse, error) {
	if s.files == nil || s.codec == nil {
		return nil, status.Error(codes.Unavailable, "document api is not configured")
	}
	obj, err := s.files.GetObject(s.fileCtx(ctx), &filesv1.GetObjectRequest{Meta: req.GetMeta(), Id: req.GetFileId()})
	if err != nil {
		return nil, err
	}
	book, err := s.codec.Load(obj.GetBody())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	objects, err := preview.Encode(preview.Capture(book))
	if err != nil {
		return nil, err
	}
	for _, object := range objects {
		key := fmt.Sprintf("jobs/%s/preview/%s/%s", req.GetJobId(), req.GetFileId(), object.Name)
		if _, err := s.files.PutObject(s.fileCtx(ctx), &filesv1.PutObjectRequest{
			Meta: req.GetMeta(), Key: key, Name: object.Name, ContentType: object.ContentType, Body: object.Content,
		}); err != nil {
			return nil, err
		}
	}
	return &documentsv1.BuildPreviewResponse{SnapshotId: req.GetFileId()}, nil
}
