# gateway-service

The only externally reachable backend. Public HTTP on `GATEWAY_ADDR`.

Job reports (`GET /api/v1/jobs/{id}/report`) expose canonical `category` and
`match_reasons` on each row. `status` remains for older clients. Summary
counters `needs_decision`, `not_in_source`, `check_name_or_volume`,
`not_in_blank`, `to_order`, and `order_not_needed` are the same in `standard`
and `smart` matching mode. See `api/openapi.yaml`.
