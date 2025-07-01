FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod and sum files (if they exist)
COPY go.mod ./
COPY go.sum* ./

# Initialize or update dependencies
RUN go mod tidy
RUN go mod download

COPY . .
RUN go build -o mock-weather-server .

FROM alpine:latest
RUN apk --no-cache add ca-certificates curl
WORKDIR /root/

COPY --from=builder /app/mock-weather-server .

EXPOSE 8080

CMD ["./mock-weather-server"]
