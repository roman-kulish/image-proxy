FROM golang:1.11.0-alpine
MAINTAINER Roman Kulish <roman.kulish@news.com.au>

RUN set -xe && \
    apk --no-cache add ca-certificates git && \
    go get -u github.com/roman-kulish/image-proxy

WORKDIR /go/src/github.com/roman-kulish/image-proxy/cmd/roxy

RUN set -xe && \
    go install -i -ldflags="-s -w" .

EXPOSE 8080

CMD ["roxy"]
