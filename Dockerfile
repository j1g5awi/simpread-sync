FROM golang:1.18-alpine AS builder

RUN go env -w GO111MODULE=auto \
    && go env -w CGO_ENABLED=0 \
    && go env -w GOPROXY=https://goproxy.cn,direct 

WORKDIR /build

COPY ./ .

RUN set -ex \
    && cd /build \
    && go build -ldflags "-s -w -extldflags '-static'" -o simpread-sync

FROM alpine:latest

COPY --from=builder /build/simpread-sync /usr/bin/simpread-sync
RUN chmod +x /usr/bin/simpread-sync

WORKDIR /data

ENTRYPOINT [ "/usr/bin/simpread-sync" ]