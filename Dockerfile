FROM golang:1.19-alpine AS builder
RUN  sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories

RUN apk add --no-cache git \
  && go env -w GO111MODULE=auto \
  && go env -w CGO_ENABLED=1 \
  && go env -w GOPROXY=https://goproxy.cn,direct

WORKDIR /build

COPY ./ .

RUN set -ex \
    && BUILD=`date +%FT%T%z` \
    && COMMIT_SHA1=`git rev-parse HEAD` \
    && go build -ldflags "-s -w -extldflags '-static' -X main.Version=${COMMIT_SHA1}|${BUILD}"  -o ck_selenium


FROM alpine AS production

RUN  sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories

RUN apk add --no-cache tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone


ENV SELENIUM_CHROME_ADDR "http://127.0.0.1:4444/wd/hub"

COPY --from=builder /build/ck_selenium /usr/bin/ck_selenium

WORKDIR /data

EXPOSE 9999

ENTRYPOINT [ "/usr/bin/ck_selenium" ]