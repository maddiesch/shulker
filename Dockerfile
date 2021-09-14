FROM alpine:3.14

RUN wget -O /etc/apk/keys/amazoncorretto.rsa.pub https://apk.corretto.aws/amazoncorretto.rsa.pub && \
    echo "https://apk.corretto.aws/" >> /etc/apk/repositories && apk update

RUN apk add amazon-corretto-16

RUN mkdir -p /data/minecraft

WORKDIR /data

ARG minecraft_version
ARG purpur_build

ENV JAVA_COMMAND /usr/bin/java
ENV JAVA_ARGUMENTS ""
ENV MINECRAFT_DIR /data/minecraft
ENV SERVER_URL "https://api.pl3x.net/v2/purpur/$minecraft_version/$purpur_build/download"
ENV ACCEPT_MOJANG_EULA false

EXPOSE 25565/tcp
EXPOSE 25565/udp
EXPOSE 25580/tcp

ADD bootstrap .
ADD server.properties .

CMD [ "/data/bootstrap" ]
