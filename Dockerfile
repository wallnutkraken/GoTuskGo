FROM golang:latest

ENV GO111MODULE on

WORKDIR /go/src/github.com/wallnutkraken/gotuskgo
COPY . .

# Set the locale and build
RUN apt update && apt install python3-pip locales -y && pip3 install textgenrnn tensorflow psutil && \
	locale-gen en_US.UTF-8 && \
	mkdir opdata && \
	go install ./cmd/gotuskgo
ENV LANG en_US.UTF-8  
ENV LANGUAGE en_US:en  
ENV LC_ALL en_US.UTF-8  

CMD ["gotuskgo"]
