FROM busybox:glibc

ADD ./kail /kail
ADD ./remote_syslog /remote_syslog
ADD ./init /init
RUN chmod +x /init

ENTRYPOINT ["./init"]
CMD [""]

