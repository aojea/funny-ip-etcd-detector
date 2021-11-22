ARG GOARCH="amd64"
# STEP 1: Build binary
FROM golang:1.17 AS builder
# golang envs
ARG GOARCH="amd64"
ARG GOOS=linux
ENV CGO_ENABLED=0
ENV GO111MODULE="on"
# copy in sources
WORKDIR /src
COPY . .
# build
RUN CGO_ENABLED=0 go build -o /go/bin/funny-ip-etcd-detector

# STEP 2: Build small image
FROM k8s.gcr.io/etcd:3.5.1-0
CMD ["/bin/funny-ip-etcd-detector"]