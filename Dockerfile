FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/server .
COPY backend/migration/ ./migration/
EXPOSE 8080
CMD ["./server"]
