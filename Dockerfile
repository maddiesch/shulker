FROM golang:1.17-alpine3.14 as ctl-build

WORKDIR /go/src

ADD shulker-ctl/go.mod shulker-ctl/go.sum ./

RUN go mod download

ADD shulker-ctl .

RUN go build -o ../bin .

FROM alpine:3.14

RUN wget -O /etc/apk/keys/amazoncorretto.rsa.pub https://apk.corretto.aws/amazoncorretto.rsa.pub && \
    echo "https://apk.corretto.aws/" >> /etc/apk/repositories && apk update

RUN apk add amazon-corretto-17

RUN mkdir -p /shulker/data

WORKDIR /shulker

RUN mkdir -p /shulker/bin

ARG minecraft_version
ARG purpur_build

ENV JAVA_COMMAND /usr/bin/java
ENV JAVA_ARGUMENTS ""
ENV MINECRAFT_DIR /shulker/data
ENV SERVER_URL "https://api.pl3x.net/v2/purpur/$minecraft_version/$purpur_build/download"
ENV ACCEPT_MOJANG_EULA false
ENV PATH "$PATH:/shulker/bin"

EXPOSE 25565/tcp
EXPOSE 25565/udp
EXPOSE 25580/tcp

COPY --from=ctl-build /go/bin /shulker/bin

ADD bootstrap .
ADD server.properties .

CMD [ "/shulker/bootstrap" ]
