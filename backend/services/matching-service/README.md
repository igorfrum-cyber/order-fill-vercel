# matching-service

Internal gRPC service. Health HTTP on `MATCHING_HEALTH_ADDR`, gRPC on `MATCHING_GRPC_ADDR`.

The service accepts already-parsed structured items (`article`, `name`, `volume`, `form`, ЧЗ flag) and returns a canonical `ReportCategory` plus structured `MatchReasons`. It does not read Excel workbooks or object storage. `document-service` maps workbook rows to items, then applies the returned identity decisions.
