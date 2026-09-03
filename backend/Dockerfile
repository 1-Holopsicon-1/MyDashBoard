FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget su-exec
WORKDIR /app
COPY --from=build /server ./server
COPY docker-entrypoint.sh /docker-entrypoint.sh
RUN adduser -D -u 10001 app && mkdir -p /app/data && chown -R app:app /app && chmod +x /docker-entrypoint.sh
EXPOSE 8081
ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["./server"]
