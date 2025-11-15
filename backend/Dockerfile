FROM golang:1.25.1 as dev

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

from dev as live-dev

RUN curl -sSfL https://raw.githubusercontent.com/air-verse/air/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

CMD ["air", "-c", "/app/.air.toml"]

FROM dev as final

RUN CGO_ENABLED=0 GOOS=linux go build  -o /app/api ./avenuectl

CMD ["/app/api"]
