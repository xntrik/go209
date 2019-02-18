FROM golang:1.11-alpine AS builder
MAINTAINER Christian Frichot <xntrik@gmail.com>

RUN apk add bash ca-certificates git gcc g++ libc-dev libgcc make
WORKDIR /go/src/github.com/xntrik/go209
COPY . .
RUN make clean
RUN go get
RUN STATIC_BUILD=1 make buildplugins
RUN make static

FROM alpine AS go209
RUN apk add ca-certificates
WORKDIR /app
COPY --from=builder /go/src/github.com/xntrik/go209/go209 /bin/go209
COPY --from=builder /go/src/github.com/xntrik/go209/*.so /app/
COPY rules.json /app/rules.json
ENTRYPOINT ["/bin/go209"]

