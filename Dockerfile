FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o gopenclaw ./cmd/gopenclaw/

FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /root/

COPY --from=builder /app/gopenclaw .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./gopenclaw"]
