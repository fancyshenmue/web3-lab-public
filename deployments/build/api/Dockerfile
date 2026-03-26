FROM golang:alpine AS builder

WORKDIR /app
COPY backend/go.mod backend/go.sum* ./
RUN go mod download || true
COPY backend/ .
RUN CGO_ENABLED=0 go build -o /api ./cmd/api

FROM alpine:latest
WORKDIR /app
COPY --from=builder /api /app/api
EXPOSE 8080
CMD ["/app/api", "serve"]
