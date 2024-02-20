FROM golang:1.22-alpine as builder

WORKDIR /
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

FROM alpine:3.18
RUN apk add --no-cache ca-certificates curl
COPY --from=builder /main /main
CMD ["/main"]
