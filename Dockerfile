FROM alpine
COPY acoustid-priv-api /usr/local/bin/
CMD ["acoustid-priv-api"]