# syntax=mirror.gcr.io/docker/dockerfile:1
# Build context: backend/
FROM mirror.gcr.io/library/golang:1.26-alpine AS build
WORKDIR /src
ENV GOPROXY=https://proxy.golang.org,https://goproxy.io,direct
COPY pkg ./pkg
COPY proto ./proto
COPY services/calculation-service ./services/calculation-service
WORKDIR /src/services/calculation-service
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -o /out/app ./cmd/calculation

FROM mirror.gcr.io/library/alpine:3.22
RUN apk add --no-cache wget
RUN adduser -D -u 10001 app
COPY --from=build /out/app /usr/local/bin/app
USER app
CMD ["app"]
