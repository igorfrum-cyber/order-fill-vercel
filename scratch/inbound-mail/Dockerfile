# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod main.go ./
RUN CGO_ENABLED=0 go build -o /out/inbound .

FROM alpine:3.22
RUN adduser -D -u 10001 app
COPY --from=build /out/inbound /usr/local/bin/inbound
USER app
ENV INBOUND_ADDR=:8787 INBOUND_DIR=/data
VOLUME ["/data"]
EXPOSE 8787
CMD ["inbound"]
