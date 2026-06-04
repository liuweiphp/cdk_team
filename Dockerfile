FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

FROM mcr.microsoft.com/playwright/python:v1.56.0-noble
WORKDIR /app
COPY backend/automation/requirements.txt ./automation/requirements.txt
RUN pip install --no-cache-dir -r ./automation/requirements.txt
COPY --from=builder /app/server .
COPY backend/migration/ ./migration/
COPY backend/automation/ ./automation/
EXPOSE 8080
CMD ["./server"]
