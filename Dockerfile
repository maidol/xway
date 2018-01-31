# FROM scratch
# ADD ca-certificates.crt /etc/ssl/certs/  # when use https, resolve Go’s x509 error
FROM alpine:latest
ADD app /
ADD conf /conf
EXPOSE 9799
EXPOSE 9788
CMD ["/app"]

# golang:onbuild automatically copies the package source, 
# fetches the application dependencies, builds the program, 
# and configures it to run on startup 
# FROM golang:onbuild
# LABEL Name=cw-gateway Version=0.0.1 
# EXPOSE 9799

# For more control, you can copy and build manually
# FROM golang:latest 
# LABEL Name=xway Version=0.0.1 
# RUN mkdir -p /go/src/xway
# ADD . /go/src/xway
# WORKDIR /go/src/xway
# ENV CGO_ENABLED 0
# RUN go build -a -installsuffix cgo -o app .
# EXPOSE 9799
# EXPOSE 9788
# CMD ["/go/src/xway/app"]
