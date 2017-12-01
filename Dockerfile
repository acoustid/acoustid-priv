FROM alpine
RUN apk --no-cache add ca-certificates
COPY acoustid-priv-api /usr/local/bin/
CMD ["acoustid-priv-api"]
