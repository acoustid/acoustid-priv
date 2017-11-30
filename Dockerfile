FROM golang:1.9-alpine
WORKDIR /opt/acoustid
COPY acoustid-priv-api .
CMD ["/opt/acoustid/acoustid-priv-api"]
