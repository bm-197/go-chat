FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/server

FROM golang:1.24-alpine AS development

WORKDIR /app

RUN go install github.com/air-verse/air@latest

COPY .air.toml ./

COPY go.mod go.sum ./
RUN go mod download

COPY . .
CMD ["air"]

FROM alpine:latest AS production

WORKDIR /app

COPY --from=builder /app/main .

RUN adduser -D appuser
USER appuser

CMD ["./main"]