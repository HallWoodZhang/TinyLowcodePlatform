FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/ts-quickjs ./cmd/ts-quickjs

FROM alpine:3.21

WORKDIR /app
RUN mkdir -p cmd/ts-quickjs/conf
COPY --from=builder /bin/ts-quickjs /usr/local/bin/
COPY cmd/ts-quickjs/conf/config.json.sample cmd/ts-quickjs/conf/config.json

EXPOSE 9720
CMD ["ts-quickjs"]
