FROM ubuntu:16.04

RUN  mkdir -p /data/app/log
COPY web_release.conf /data/app/web.conf
COPY damogo /data/app

ENV PATH /data/app:$PATH
EXPOSE 8009
WORKDIR /data/app
CMD ["damogo"]