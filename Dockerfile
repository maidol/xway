FROM scratch
ADD app /
EXPOSE 9799
CMD ["/app"]

# golang:onbuild automatically copies the package source, 
# fetches the application dependencies, builds the program, 
# and configures it to run on startup 
# FROM golang:onbuild
# LABEL Name=cw-gateway Version=0.0.1 
# EXPOSE 9799

# For more control, you can copy and build manually
# FROM golang:latest 
# LABEL Name=cw-gateway Version=0.0.1 
# RUN mkdir /app 
# ADD . /app/ 
# WORKDIR /app 
# RUN go build -o main .
# EXPOSE 3000 
# CMD ["/app/main"]
