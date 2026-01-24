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

# Create a non-root user and group
RUN addgroup -S app && adduser -S app -G app

USER app

CMD ["/app/api"]
