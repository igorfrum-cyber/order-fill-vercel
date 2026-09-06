package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"order-fill/backend/pkg/grpcutil"
	"order-fill/backend/pkg/healthz"
	documentsv1 "order-fill/backend/proto/gen/go/orderfill/documents/v1"
	filesv1 "order-fill/backend/proto/gen/go/orderfill/files/v1"
	jobsv1 "order-fill/backend/proto/gen/go/orderfill/jobs/v1"
	"order-fill/backend/services/document-service/internal/adapter/inbound/queue"
	"order-fill/backend/services/document-service/internal/adapter/outbound/grpcjobs"
	"order-fill/backend/services/document-service/internal/adapter/outbound/xlsx"
	"order-fill/backend/services/document-service/internal/app/usecase"
	"order-fill/backend/services/document-service/internal/clients/brand"
	"order-fill/backend/services/document-service/internal/clients/calculation"
	"order-fill/backend/services/document-service/internal/clients/matching"
	"order-fill/backend/services/document-service/internal/config"
	"order-fill/backend/services/document-service/internal/domain/orderfill"
	"order-fill/backend/services/document-service/internal/transport/grpcapi"
)

func HealthHandler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthz.Live())
	mux.Handle("GET /readyz", healthz.Ready(nil))
	return mux
}

func Run(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	var handler documentsv1.DocumentServiceServer
	if cfg.FileAddr != "" {
		if cfg.BrandAddr == "" {
			return fmt.Errorf("BRAND_GRPC_ADDR is required")
		}
		fileConn, err := grpcutil.Dial(ctx, cfg.FileAddr)
		if err != nil {
			return err
		}
		brands, err := brand.Dial(ctx, cfg.BrandAddr)
		if err != nil {
			return err
		}
		handler = grpcapi.NewServer(filesv1.NewFileServiceClient(fileConn), xlsx.NewCodec(), brands)
	} else {
		handler = grpcapi.NewServer(nil, nil, nil)
	}
	return grpcutil.Serve(ctx, cfg.GRPCAddr, cfg.HealthAddr, grpcapi.New(handler), HealthHandler())
}

func RunWorker(ctx context.Context, cfg config.Config, log *slog.Logger) error {
	fileConn, err := grpcutil.Dial(ctx, cfg.FileAddr)
	if err != nil {
		return err
	}
	jobConn, err := grpcutil.Dial(ctx, cfg.JobAddr)
	if err != nil {
		return err
	}
	filesAPI := filesv1.NewFileServiceClient(fileConn)
	store := grpcjobs.Files{API: filesAPI}
	jobsAPI := grpcjobs.Jobs{API: jobsv1.NewJobServiceClient(jobConn), Files: store}
	reports := grpcjobs.Reports{Files: store}
	if cfg.CalculationAddr == "" {
		return fmt.Errorf("CALCULATION_GRPC_ADDR is required")
	}
	calc, err := calculation.Dial(ctx, cfg.CalculationAddr)
	if err != nil {
		return err
	}
	if cfg.MatchingAddr == "" {
		return fmt.Errorf("MATCHING_GRPC_ADDR is required")
	}
	matchClient, err := matching.Dial(ctx, cfg.MatchingAddr)
	if err != nil {
		return err
	}
	if cfg.BrandAddr == "" {
		return fmt.Errorf("BRAND_GRPC_ADDR is required")
	}
	brands, err := brand.Dial(ctx, cfg.BrandAddr)
	if err != nil {
		return err
	}
	var match orderfill.Matcher = matchClient
	processor := usecase.NewProcessJob(xlsx.NewCodec(), store, jobsAPI, reports, time.Now, log, nil, calc, match, brands)
	consumer, err := queue.NewConsumer(cfg.QueueURL, "", log)
	if err != nil {
		return err
	}
	httpSrv := &http.Server{Addr: cfg.HealthAddr, Handler: HealthHandler(), ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = httpSrv.ListenAndServe() }()
	log.Info("document-worker consuming", "queue", cfg.QueueURL)
	err = consumer.Run(ctx, processor.Handle)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutdownCtx)
	return err
}
