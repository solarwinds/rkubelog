# Copyright 2019 SolarWinds Worldwide, LLC.
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.16.0-alpine as main
RUN apk update && apk add --no-cache git ca-certificates wget && update-ca-certificates
RUN wget -O /etc/ssl/certs/papertrail-bundle.pem https://papertrailapp.com/tools/papertrail-bundle.pem
WORKDIR /github.com/solarwinds/rkubelog
ADD . .
RUN CGO_ENABLED=0 go build -mod vendor -ldflags='-w -s -extldflags "-static"' -a -o /rkubelog .

FROM alpine:3.13.2
COPY --from=main /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=main /etc/ssl/certs/papertrail-bundle.pem /etc/ssl/certs/
COPY --from=main /rkubelog /app/rkubelog
WORKDIR /app
RUN mkdir db; chmod -R 777 db
USER 1001
ENTRYPOINT ["/app/rkubelog"]
