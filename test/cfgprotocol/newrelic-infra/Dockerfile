ARG GOLANG_VERSION=1.16
ARG base_image=alpine:3.13

FROM golang:${GOLANG_VERSION} as builder
WORKDIR /code
COPY go.mod .
RUN go mod download

COPY . ./


RUN go build -gcflags "all=-N -l" -o /usr/bin/newrelic-infra cmd/newrelic-infra/newrelic-infra.go

# Compile Delve
RUN go get github.com/go-delve/delve/cmd/dlv


FROM $base_image AS core

# Add the agent binary
COPY --from=builder /usr/bin/newrelic-infra /usr/bin/newrelic-infra
COPY --from=builder /go/bin/dlv /

RUN apk --no-cache upgrade

RUN apk add --no-cache --upgrade \
    ca-certificates=20191127-r5 \
    && mkdir /lib64 \
    && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2 \
    && apk add --no-cache tini=0.19.0-r0 \
    && apk add jq