FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN go build -o /out/worker ./cmd/worker

FROM alpine:3.22
COPY --from=build /out/worker /usr/local/bin/worker
EXPOSE 8081
CMD ["worker"]
