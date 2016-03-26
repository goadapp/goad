FROM golang:1.5

RUN apt-get update
RUN apt-get install -y zip
ADD . /go/src/github.com/gophergala2016/goad
WORKDIR /go/src/github.com/gophergala2016/goad
RUN go get -u github.com/jteeuwen/go-bindata/... 
RUN make bindata
RUN go build -o /go/bin/goad-api webapi/webapi.go

CMD ["/go/bin/goad-api", "-addr", ":8080"]
EXPOSE 8080
