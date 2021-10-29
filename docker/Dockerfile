FROM public.ecr.aws/bitnami/golang:1.17 as go-build

WORKDIR /go/src

FROM go-build as ctl-build

ADD ./shulker-ctl .

RUN go build -o ../bin .

FROM go-build as box-build

ADD ./shulker-box .

RUN go build -o ../bin .

FROM amazonlinux:2

RUN rpm --import https://yum.corretto.aws/corretto.key
RUN curl -L -o /etc/yum.repos.d/corretto.repo https://yum.corretto.aws/corretto.repo
RUN yum install -y java-17-amazon-corretto-devel

RUN mkdir -p /shulker/bin
RUN mkdir -p /shulker/data

WORKDIR /shulker

ENV PATH $PATH:/shulker/bin

COPY --from=ctl-build /go/bin ./bin
COPY --from=box-build /go/bin ./bin
COPY docker/config.shulker.hcl ./data/config.shulker.hcl

CMD ["shulker-box", "-config", "./data/config.shulker.hcl", "-log", "./data/shulker-logs/shulker.log", "-loglevel", "trace"]
