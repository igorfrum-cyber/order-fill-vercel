# document-service

`cmd/document-api` (gRPC) and `cmd/document-worker` (queue consumer).

Workbook parsing and writing stay in this service. Product identity comes from `matching-service`: the worker maps parsed rows to structured items, sends them over gRPC, and maps returned categories/reasons into `report.json`. Quantity math stays in `calculation-service`.
