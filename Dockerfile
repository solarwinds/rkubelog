FROM golang:1.15.0-alpine as main
RUN apk update && apk add --no-cache git ca-certificates wget && update-ca-certificates
RUN wget -O /etc/ssl/certs/papertrail-bundle.pem https://papertrailapp.com/tools/papertrail-bundle.pem
WORKDIR /github.com/solarwinds/rkubelog
ADD . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-w -s -extldflags "-static"' -a -o /rkubelog .

FROM alpine
COPY --from=main /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=main /etc/ssl/certs/papertrail-bundle.pem /etc/ssl/certs/
COPY --from=main /rkubelog /app/rkubelog
RUN chmod -R 777 /app
USER 1001
WORKDIR /app
ENTRYPOINT ./rkubelog