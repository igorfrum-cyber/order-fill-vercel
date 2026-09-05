package bootstrap

import (
	"context"
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
	"order-fill/backend/services/document-service/internal/clients/calculation"
	"order-fill/backend/services/document-service/internal/config"
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
		conn, err := grpcutil.Dial(ctx, cfg.FileAddr)
		if err != nil {
			return err
		}
		handler = grpcapi.NewServer(filesv1.NewFileServiceClient(conn), xlsx.NewCodec())
	} else {
		handler = grpcapi.NewServer(nil, nil)
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
	var calc calculation.Client
	if cfg.CalculationAddr != "" {
		client, err := calculation.Dial(ctx, cfg.CalculationAddr)
		if err != nil {
			return err
		}
		calc = client
	}
	processor := usecase.NewProcessJob(xlsx.NewCodec(), store, jobsAPI, reports, time.Now, log, nil, calc)
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
