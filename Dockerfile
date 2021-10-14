FROM golang:alpine as builder

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
	GOPROXY="https://goproxy.cn,direct"
	
WORKDIR /sd_etcd

COPY . .

RUN go build

FROM alpine:latest
RUN apk --no-cache add ca-certificates && \
    rm -rf /var/cache/apk/* /tmp/*
WORKDIR /dist
COPY --from=builder /sd_etcd/sd_etcd .
EXPOSE 80
CMD ["/dist/sd_etcd"]
