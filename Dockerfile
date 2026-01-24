FROM golang:1.25.1-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

from builder as live-dev

RUN apk add --no-cache curl

RUN curl -sSfL https://raw.githubusercontent.com/air-verse/air/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

CMD ["air", "-c", "/app/.air.toml"]

FROM builder AS image

RUN CGO_ENABLED=0 GOOS=linux go build -tags prod -o /app/api ./

FROM alpine:latest as final

WORKDIR /app
COPY --from=image /app/api /app/api

# Install su-exec for dropping privileges
RUN apk add --no-cache su-exec

# Copy the entrypoint script
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Create a non-root user and group
RUN addgroup -S app && adduser -S app -G app

ENTRYPOINT ["/entrypoint.sh"]

CMD ["/app/api"]
