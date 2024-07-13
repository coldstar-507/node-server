# syntax=docker/dockerfile:1

FROM golang:1.22.1

WORKDIR /app

COPY . .

RUN go mod download

# RUN go test ./tests/test_test.go -v

# RUN CGO_ENABLED=0 GOOS=linux go build -C ./cmd/custom-back/ -o ./bin/main

EXPOSE 8080

CMD ["go", "test", "./tests/test_test.go", "-v"]
