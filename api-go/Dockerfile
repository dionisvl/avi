ARG GO_VERSION=1.26.2
ARG GO_ALPINE_TAG=alpine3.23
ARG ALPINE_VERSION=3.23.3

FROM golang:${GO_VERSION}-${GO_ALPINE_TAG} AS dev
RUN apk add --no-cache git bash curl ripgrep make jq htop
RUN go install github.com/air-verse/air@v1.65.0 \
    && go install github.com/swaggo/swag/cmd/swag@v1.16.6 \
    && go install mvdan.cc/gofumpt@v0.9.0 \
    && go install github.com/daixiang0/gci@v0.14.0 \
    && go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.4 \
    && go install golang.org/x/tools/go/analysis/passes/modernize/cmd/modernize@v0.44.0
WORKDIR /app
COPY . .
RUN go mod download
EXPOSE 8080
CMD ["air", "-c", ".air.toml"]


FROM golang:${GO_VERSION}-${GO_ALPINE_TAG} AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG APP_VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-X 'github.com/dionisvl/avi/api-go/internal/config.Version=${APP_VERSION}'" \
    -o /bin/avi-api ./cmd/api

FROM alpine:${ALPINE_VERSION} AS runtime
RUN apk add --no-cache ca-certificates \
    && addgroup -S appgroup \
    && adduser -S appuser -G appgroup
COPY --from=builder /bin/avi-api /bin/avi-api
USER appuser
EXPOSE 8080
ENTRYPOINT ["/bin/avi-api"]

FROM runtime AS prod
