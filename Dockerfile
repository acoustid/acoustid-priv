FROM go:1.9-alpine
WORKDIR /opt/acoustid/priv
COPY acoustid-priv-api .
CMD ["/opt/acoustid/priv/acoustid-priv-api"]
