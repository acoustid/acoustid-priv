FROM alpine
RUN apk --no-cache add curl ca-certificates
EXPOSE 3382
HEALTHCHECK CMD curl -f http://localhost:3382/_health || exit 1
COPY acoustid-priv-api /usr/local/bin/
CMD ["acoustid-priv-api"]
