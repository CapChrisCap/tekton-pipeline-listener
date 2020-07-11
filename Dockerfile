FROM golang:1.14 as cli-builder
WORKDIR /go/src/github.com/capchriscap/tekton-cli
RUN git clone https://github.com/capchriscap/tekton-cli .
RUN export GO111MODULE=on && make bin/tkn

FROM golang:1.14 as server-builder
WORKDIR /go/src/github.com/capchriscap/tekton-pipeline-listener
ENV GO111MODULE=on
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY main.go .
RUN GOOS=linux GOARCH=386 go build -o webserver main.go

FROM busybox
COPY --from=cli-builder /go/src/github.com/capchriscap/tekton-cli/bin/tkn /usr/local/bin/tkn
COPY --from=server-builder /go/src/github.com/capchriscap/tekton-pipeline-listener/webserver /usr/local/bin/webserver
CMD webserver
