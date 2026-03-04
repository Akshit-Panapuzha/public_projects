FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -o /app .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=build /app /app
EXPOSE 8080
ENTRYPOINT ["/app"]
