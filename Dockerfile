FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o server ./cmd/main.go

FROM alpine:3.18

WORKDIR /app

COPY --from=builder /app/ .
RUN mkdir -p uploads

COPY config.yaml .

EXPOSE 8080

CMD ["./server"]
