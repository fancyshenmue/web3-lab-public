FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY cmd/mock-price/ cmd/mock-price/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /mock-price ./cmd/mock-price

FROM alpine:3.20
RUN apk --no-cache add ca-certificates
COPY --from=builder /mock-price /usr/local/bin/mock-price
EXPOSE 4050
ENTRYPOINT ["mock-price"]
