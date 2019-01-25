FROM golang:latest

ENV GO111MODULE on

WORKDIR /go/src/github.com/wallnutkraken/gotuskgo
COPY . .

RUN mkdir opdata && \
	go install ./cmd/gotuskgo

CMD ["gotuskgo"]