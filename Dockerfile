FROM golang:1.14 as main
RUN adduser --disabled-login appuser
WORKDIR /github.com/solarwinds/cabbage
ADD . .
RUN go build -ldflags="-w -s" -a -o /cabbage .

FROM alpine
RUN apk --update add ca-certificates
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
COPY --from=main /cabbage /app/cabbage
COPY --from=main /etc/passwd /etc/passwd
RUN chmod -R 777 /app
USER appuser
WORKDIR /app
ENTRYPOINT ./cabbage