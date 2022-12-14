#!/bin/bash
#docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
docker buildx create --use --name mybuilder
docker buildx build -t scjtqs/jd_cookie:selenium -f Dockerfile --platform linux/amd64,linux/arm64,linux/arm/v7 --push .
docker buildx rm mybuilder
