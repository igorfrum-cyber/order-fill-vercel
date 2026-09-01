# syntax=mirror.gcr.io/docker/dockerfile:1
FROM mirror.gcr.io/library/golang:1.26-alpine AS build

WORKDIR /src
ENV GOPROXY=https://proxy.golang.org,https://goproxy.io,direct
ENV GODEBUG=http2client=0

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    sh -eu -c 'i=1; while [ "$i" -le 5 ]; do go mod download && exit 0; echo "go mod download failed (attempt $i)"; i=$((i+1)); sleep 4; done; exit 1'
COPY cmd ./cmd
COPY internal ./internal
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -o /out/worker ./cmd/worker

FROM mirror.gcr.io/library/alpine:3.22
RUN adduser -D -u 10001 app
COPY --from=build /out/worker /usr/local/bin/worker
USER app
EXPOSE 8081
CMD ["worker"]
