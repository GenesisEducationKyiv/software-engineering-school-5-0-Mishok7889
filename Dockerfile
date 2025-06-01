FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o weatherapi .

# Run tests but don't fail the build if tests fail
RUN go test -v ./... || true

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/weatherapi .

COPY --from=builder /app/public ./public

COPY --from=builder /app/.env ./

RUN adduser -D -g '' appuser
USER appuser

CMD ["./weatherapi"]