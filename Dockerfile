FROM golang:1.27 AS build_base

## Creates a docker image with Federation. Requires a redis server connection
##
## Usage:
## docker build . -t federation
## docker run -d -v config:/app/config -p 18900:18900 federation

RUN mkdir /compile
  
COPY . /compile  

RUN cd /compile  && CGO_ENABLED=0 go build -o /compile/BarcodeServer BarcodeServer/cmd/barcodeserver

FROM alpine:3.24


RUN apk add ca-certificates redis && \
   mkdir /app
  
COPY --from=build_base /compile/BarcodeServer /app/FederationServer

WORKDIR /app/
CMD ["/app/FederationServer"]


