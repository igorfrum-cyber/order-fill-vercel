FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -o /out/worker ./cmd/worker

FROM alpine:3.22
RUN adduser -D -u 10001 app
COPY --from=build /out/worker /usr/local/bin/worker
USER app
EXPOSE 8081
CMD ["worker"]
