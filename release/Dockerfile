FROM golang:1.15

## To compile:
## cd barcodebuddy-federation/release/
## docker build . --tag federationrelease
## docker run --rm -it -v ../:/usr/src/myapp -w /usr/src/myapp federationrelease

RUN \
  apt-get update && \
  apt-get install -y ca-certificates openssl zip && \
  update-ca-certificates && \
  rm -rf /var/lib/apt

COPY entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
