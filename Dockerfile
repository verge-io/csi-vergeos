# Build stage
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /csi-vergeos ./cmd/csi-vergeos/

# Runtime stage
FROM alpine:3.21

# NFS and mount utilities needed by the node plugin.
RUN apk add --no-cache \
    nfs-utils \
    e2fsprogs \
    blkid \
    util-linux

COPY --from=builder /csi-vergeos /usr/local/bin/csi-vergeos

ENTRYPOINT ["/usr/local/bin/csi-vergeos"]
