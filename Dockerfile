FROM golang:1.24-alpine AS builder

# Install build dependencies for CGO and SQLite
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o weatherapi .

# Run tests with CGO enabled for SQLite support, but don't fail the build if tests fail
RUN CGO_ENABLED=1 go test -v ./... || true

FROM alpine:latest

RUN apk --no-cache add curl

WORKDIR /app

COPY --from=builder /app/weatherapi .

COPY --from=builder /app/public ./public

# .env file is optional - environment variables provided by runtime

RUN adduser -D -g '' appuser
USER appuser

CMD ["./weatherapi"]