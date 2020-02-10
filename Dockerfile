FROM alpine:latest

ADD ./kail /kail
ADD ./init /init

RUN apk add -U util-linux
RUN chmod +x /init

ENTRYPOINT ["./init"]
CMD [""]

