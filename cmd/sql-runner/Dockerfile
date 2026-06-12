FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/sql-runner ./cmd/sql-runner

FROM alpine:3.21

WORKDIR /app
RUN mkdir -p cmd/sql-runner/conf
COPY --from=builder /bin/sql-runner /usr/local/bin/
COPY cmd/sql-runner/conf/config.json.sample cmd/sql-runner/conf/config.json

EXPOSE 9721
CMD ["sql-runner"]
