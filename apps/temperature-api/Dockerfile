FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod main.go ./
RUN CGO_ENABLED=0 go build -trimpath -o /temperature-api .

FROM alpine:3.23
COPY --from=builder /temperature-api /temperature-api
EXPOSE 8081
CMD ["/temperature-api"]
