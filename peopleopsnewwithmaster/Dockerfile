FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/peopleops ./cmd/server
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata && addgroup -S peopleops && adduser -S -G peopleops -h /app peopleops
WORKDIR /app
COPY --from=build /out/peopleops /app/peopleops
COPY scripts/healthcheck.sh /app/peopleops-healthcheck
RUN chmod +x /app/peopleops-healthcheck
USER peopleops
ENV HTTP_ADDR=:8080 APP_ENV=production
EXPOSE 8080
ENTRYPOINT ["/app/peopleops"]
