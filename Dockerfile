FROM golang:1.15.0-alpine

RUN mkdir -p /gocompositor-examples
ADD . /gocompositor-examples
WORKDIR /gocompositor-examples

ENV CGO_ENABLED=1

RUN apk update && apk add gstreamer \
                gst-plugins-base \
                gst-plugins-bad \
                gst-plugins-good \
                gst-plugins-ugly \
                gst-plugins-base-dev \
                gst-plugins-bad-dev \
                pkgconfig \
                build-base \
                git

VOLUME /gocompositor-examples

RUN go mod download

ENTRYPOINT ["go", "run" ]