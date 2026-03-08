FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/api

FROM alpine:latest
RUN addgroup -g 1000 appgroup && adduser -D -u 1000 -G appgroup appuser
WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /app/migrations ./migrations
USER appuser
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s CMD wget --spider http://localhost:8080/health || exit 1
CMD ["./main"]
