FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal
RUN go build -o /out/api ./cmd/api

FROM alpine:3.22
COPY --from=build /out/api /usr/local/bin/api
EXPOSE 8080
CMD ["api"]
