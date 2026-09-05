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
	"order-fill/backend/services/document-service/internal/domain/orderfill"
	"order-fill/backend/services/document-service/internal/domain/preview"
	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
)

type Server struct {
	documentsv1.UnimplementedDocumentServiceServer
	files filesv1.FileServiceClient
	codec spreadsheet.Codec
}

func NewServer(files filesv1.FileServiceClient, codec spreadsheet.Codec) *Server {
	return &Server{files: files, codec: codec}
}

func New(handler documentsv1.DocumentServiceServer) *grpc.Server {
	s := grpcutil.NewServer()
	if handler != nil {
		documentsv1.RegisterDocumentServiceServer(s, handler)
	}
	return s
}

func (s *Server) AnalyzeInputs(ctx context.Context, req *documentsv1.AnalyzeInputsRequest) (*documentsv1.AnalyzeInputsResponse, error) {
	if s.files == nil || s.codec == nil {
		return nil, status.Error(codes.Unavailable, "document api is not configured")
	}
	var brand string
	var blankNames []string
	var blankBook spreadsheet.Workbook
	for _, id := range req.GetInputFileIds() {
		obj, err := s.files.GetObject(ctx, &filesv1.GetObjectRequest{Id: id})
		if err != nil {
			return nil, err
		}
		book, err := s.codec.Load(obj.GetBody())
		if err != nil {
			continue
		}
		if detected, err := orderfill.DetectBrand(book); err == nil {
			brand = detected
			continue
		}
		blankNames = append(blankNames, obj.GetObject().GetName())
		blankBook = book
	}
	if brand == "" {
		return nil, status.Error(codes.InvalidArgument, "source workbook was not found")
	}
	plans, err := orderfill.PlanBlanks(brand, blankNames)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	blankLabel := ""
	if len(plans) > 0 {
		blankLabel = plans[0].Label
		if brand == "christina" && blankBook != nil {
			blankLabel = orderfill.LabelChristinaBlank(blankBook, blankLabel)
		}
	}
	return &documentsv1.AnalyzeInputsResponse{Brand: brand, BlankLabel: blankLabel}, nil
}

func (s *Server) BuildPreview(ctx context.Context, req *documentsv1.BuildPreviewRequest) (*documentsv1.BuildPreviewResponse, error) {
	if s.files == nil || s.codec == nil {
		return nil, status.Error(codes.Unavailable, "document api is not configured")
	}
	obj, err := s.files.GetObject(ctx, &filesv1.GetObjectRequest{Id: req.GetFileId()})
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
		if _, err := s.files.PutObject(ctx, &filesv1.PutObjectRequest{
			Key: key, Name: object.Name, ContentType: object.ContentType, Body: object.Content,
		}); err != nil {
			return nil, err
		}
	}
	return &documentsv1.BuildPreviewResponse{SnapshotId: req.GetFileId()}, nil
}
