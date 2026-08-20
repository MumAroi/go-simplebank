FROM golang:1.26-alpine3.24 AS builder
WORKDIR /app
COPY . .
RUN go build -o main main.go
RUN apk add curl
RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.19.1/migrate.linux-amd64.tar.gz | tar xvz

FROM alpine:3.24
WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /app/migrate ./migrate
COPY .env.example .env
COPY db/migration ./migration
COPY start.sh .

EXPOSE 8080
CMD ["/app/ain"]
ENTRYPOINT [ "/app/start.sh" ]
