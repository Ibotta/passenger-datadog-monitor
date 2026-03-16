FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /passenger-datadog-monitor .

FROM scratch
COPY --from=builder /passenger-datadog-monitor /usr/local/bin/passenger-datadog-monitor
ENTRYPOINT ["/usr/local/bin/passenger-datadog-monitor"]
