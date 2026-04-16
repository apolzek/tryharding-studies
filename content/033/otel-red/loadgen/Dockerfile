FROM alpine:3.20
RUN apk add --no-cache wget
COPY loadgen.sh /loadgen.sh
RUN chmod +x /loadgen.sh
CMD ["/loadgen.sh"]
