## compile proxy-init utility
FROM gcr.io/linkerd-io/go-deps:817f8e11 as golang
WORKDIR /go/src/github.com/linkerd/linkerd2
COPY ./proxy-init ./proxy-init
RUN CGO_ENABLED=0 GOOS=linux go install -v -installsuffix cgo ./proxy-init/

## package runtime
FROM gcr.io/linkerd-io/base:2017-10-30.01
COPY --from=golang /go/bin/proxy-init /usr/local/bin/proxy-init
ENTRYPOINT ["/usr/local/bin/proxy-init"]
