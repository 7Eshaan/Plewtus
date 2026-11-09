FROM golang:1.27.3-alpine AS build

WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/plewtus ./cmd/plewtus

FROM alpine:3.22
RUN apk add --no-cache ca-certificates wget \
    && addgroup -S plewtus \
    && adduser -S -G plewtus plewtus
COPY --from=build /out/plewtus /usr/local/bin/plewtus
USER plewtus
ENV PLEWTUS_HOST=0.0.0.0 \
    PORT=8080 \
    PLEWTUS_ENV=production
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q -O /dev/null "http://127.0.0.1:${PORT}/healthz" || exit 1
ENTRYPOINT ["/usr/local/bin/plewtus"]
