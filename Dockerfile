# Copyright 2019 SolarWinds Worldwide, LLC.
# SPDX-License-Identifier: Apache-2.0

FROM golang:1.16.4 as main
RUN wget -O /etc/ssl/certs/papertrail-bundle.pem https://papertrailapp.com/tools/papertrail-bundle.pem
WORKDIR /github.com/solarwinds/rkubelog
ADD . .
RUN CGO_ENABLED=0 go build -mod vendor -ldflags='-w -s -extldflags "-static"' -a -o /rkubelog .

FROM alpine:3.13.5
COPY --from=main /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=main /etc/ssl/certs/papertrail-bundle.pem /etc/ssl/certs/
COPY --from=main /rkubelog /app/rkubelog
WORKDIR /app
RUN touch db; chmod 777 db
USER 1001
ENTRYPOINT ["/app/rkubelog"]
