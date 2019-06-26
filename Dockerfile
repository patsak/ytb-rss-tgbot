FROM golang:1.12.6-alpine3.10 as builder

ENV GO111MODULE=on

WORKDIR /app

RUN apk --no-cache add git ca-certificates

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build

FROM alpine:3.10.0
RUN apk --no-cache add ffmpeg
COPY --from=builder /app/ytb-rss-tgbot /app/ytb-rss-tgbot
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/app/ytb-rss-tgbot"]