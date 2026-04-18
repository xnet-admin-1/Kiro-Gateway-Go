FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags='-s -w' -o /kiro-gateway ./cmd/kiro-gateway/

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /kiro-gateway /usr/local/bin/kiro-gateway
WORKDIR /root
EXPOSE 8080
ENTRYPOINT ["kiro-gateway"]
