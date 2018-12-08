FROM golang:1.11.2-alpine

ADD . /go/src/github.com/maxim-kuderko/file-listener

WORKDIR /go/src/github.com/maxim-kuderko/file-listener

RUN go install

CMD ["/go/bin/file-listener"]