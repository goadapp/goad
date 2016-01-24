web: go get -u github.com/jteeuwen/go-bindata/... && make bindata && cp -r . $GOPATH/src/github.com/gophergala2016/goad && go build -o bin/goad-api webapi/webapi.go
