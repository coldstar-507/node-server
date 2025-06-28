# Build stage
FROM golang:1.24.4 AS builder
WORKDIR /app
COPY . .
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

RUN go build -o ./bin/main ./cmd/main/main.go || exit 1

# Run stage (minimal)
FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=builder /app/bin/main ./bin/main
EXPOSE 8083
EXPOSE 12000
EXPOSE 12001
CMD ["/app/bin/main"]