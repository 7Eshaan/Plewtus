FROM golang:1.27.3-alpine AS build

WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Optional build metadata reported by /health (.git is not in the build context):
#   docker build --build-arg VERSION=1.2.3 --build-arg COMMIT=$(git rev-parse --short=12 HEAD) .
ARG VERSION=""
ARG COMMIT=""
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w ${VERSION:+-X plewtus/internal/config.Version=$VERSION} ${COMMIT:+-X plewtus/internal/config.commit=$COMMIT}" \
    -o /out/plewtus ./cmd/plewtus

FROM alpine:3.22
# poppler-utils provides pdftotext, used to extract text from PDF results.
RUN apk add --no-cache ca-certificates wget poppler-utils \
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
