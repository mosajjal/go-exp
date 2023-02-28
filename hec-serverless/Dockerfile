FROM alpine:edge
RUN apk add --no-cache ca-certificates
COPY  ./hec-serverless /hec-serverless
ENTRYPOINT ["/hec-serverless"]
