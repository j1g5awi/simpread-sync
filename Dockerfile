FROM golang:1.18-bullseye AS builder

RUN go env -w GO111MODULE=auto \
    && go env -w CGO_ENABLED=0 \
    && go env -w GOPROXY=https://goproxy.cn,direct 

WORKDIR /build

COPY ./ .

RUN set -ex \
    && cd /build \
    && go build -ldflags "-s -w -extldflags '-static'" -o simpread-sync

FROM debian:bullseye-slim

COPY --from=builder /build/simpread-sync /usr/bin/simpread-sync
RUN chmod +x /usr/bin/simpread-sync

RUN \
    set -ex && \
    apt-get update && \
    apt-get install -yq --no-install-recommends \
        pandoc fonts-noto-cjk wkhtmltopdf

WORKDIR /data

ENTRYPOINT [ "/usr/bin/simpread-sync" ]