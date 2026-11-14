FROM golang:1.26-alpine AS builder

WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/api ./cmd/api

FROM alpine:3.21
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/api /app/api

EXPOSE 8080
CMD ["/app/api"]
