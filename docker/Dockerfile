FROM alpine
COPY docker-entrypoint.sh /
COPY acoustid-priv-api /usr/local/bin/
CMD ["acoustid-priv-api"]
ENTRYPOINT ["/docker-entrypoint.sh"]