FROM golang:1.16 AS build_base

## !! NOT for production use !!
## 
## Creates a docker image with Federation and Redis for TESTING purposes
## For production use a compiled version as a standalone app.
##
## Usage:
## docker build . -t federation
## docker run -d -v config:/app/config -p 18900:18900 federation

RUN mkdir /compile
  
COPY . /compile  

RUN cd /compile  && CGO_ENABLED=0 go build -o /compile/BarcodeServer

FROM alpine:3.13


RUN apk add ca-certificates redis && \
   mkdir /app && \
   echo "redis-server --daemonize yes && /app/FederationServer" > /app/start.sh && \
   chmod +x /app/start.sh
  
COPY --from=build_base /compile/BarcodeServer /app/FederationServer

CMD ["sh","/app/start.sh"]


