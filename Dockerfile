FROM golang:1.16.2-buster

ENV GO111MODULE=on

RUN echo "deb http://www.deb-multimedia.org buster main" >> /etc/apt/sources.list
RUN wget https://www.deb-multimedia.org/pool/main/d/deb-multimedia-keyring/deb-multimedia-keyring_2016.8.1_all.deb
RUN dpkg -i deb-multimedia-keyring_2016.8.1_all.deb

RUN apt-get update && apt-get install -y \
    libvpx-dev

WORKDIR $GOPATH/src/github.com/pion/ion-avp

COPY go.mod go.sum ./
RUN cd $GOPATH/src/github.com/pion/ion-avp && go mod download

COPY pkg/ $GOPATH/src/github.com/pion/ion-avp/pkg
COPY cmd/ $GOPATH/src/github.com/pion/ion-avp/cmd

WORKDIR $GOPATH/src/github.com/pion/ion-avp/cmd/signal/grpc
RUN GOOS=linux go build -ldflags '-linkmode "external" -extldflags "-static"' -tags libvpx -a -installsuffix cgo -o /avp .

FROM alpine:3.13.0

RUN apk --no-cache add ca-certificates libvpx-dev
COPY --from=0 /avp /usr/local/bin/avp

COPY config.toml /configs/avp.toml

ENTRYPOINT ["/usr/local/bin/avp"]
CMD ["-c", "/configs/avp.toml"]
