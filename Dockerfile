# Build Chainnode in a stock Go builder container
FROM golang:1.13-alpine as builder

RUN apk add --no-cache make gcc musl-dev linux-headers

ADD . /chainnode
RUN cd /chainnode && build/env.sh go build

# Pull Chainnode into a second stage deploy alpine container
FROM alpine:latest

RUN apk add --no-cache ca-certificates
RUN mkdir /etc/chainnode

ARG CONFIG=config.yml

COPY --from=builder /chainnode/chainnode /usr/local/bin/
COPY --from=builder /chainnode/${CONFIG} /etc/chainnode/config.yml

EXPOSE 8888
ENTRYPOINT ["chainnode"]
CMD ["-c", "/etc/chainnode/config.yml"]