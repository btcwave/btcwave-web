FROM golang:1.23-alpine AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o btcwave-web ./cmd/web/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /app/btcwave-web /usr/local/bin/btcwave-web
RUN mkdir -p /var/lib/btcwave-web
EXPOSE 8080
ENTRYPOINT ["btcwave-web"]
CMD ["--listen", ":8080", "--data", "/var/lib/btcwave-web"]
