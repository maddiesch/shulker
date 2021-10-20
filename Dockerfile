# Build Shulker CTL
FROM golang:1.17-alpine3.14 as ctl-build

WORKDIR /go/src

ADD shulker-ctl/go.mod shulker-ctl/go.sum ./

RUN go mod download

ADD shulker-ctl .

RUN go build -o ../bin .

# Build Shulker Box
FROM golang:1.17-alpine3.14 as box-build

WORKDIR /go/src

ADD shulker-box/go.mod shulker-box/go.sum ./

RUN go mod download

ADD shulker-box .

RUN go build -o ../bin .

# Default Container
FROM alpine:3.14

RUN wget -O /etc/apk/keys/amazoncorretto.rsa.pub https://apk.corretto.aws/amazoncorretto.rsa.pub && \
    echo "https://apk.corretto.aws/" >> /etc/apk/repositories && apk update

RUN apk add amazon-corretto-17

RUN mkdir -p /shulker/data

WORKDIR /shulker

RUN mkdir -p /shulker/bin

ENV MINECRAFT_DIR /shulker/data
ENV PATH "$PATH:/shulker/bin"

EXPOSE 25565/tcp
EXPOSE 25580/tcp

COPY --from=ctl-build /go/bin /shulker/bin
COPY --from=box-build /go/bin /shulker/bin

ADD bootstrap .
ADD server.properties .
ADD shulker-box/config.shulker.hcl .

CMD [ "/shulker/bootstrap" ]
