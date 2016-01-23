lambda:
	GOOS=linux GOARCH=amd64 go build -o data/lambda/goad-lambda ./lambda
	zip -jr data/lambda data/lambda

bindata: lambda
	go-bindata -nocompress -pkg infrastructure -o infrastructure/bindata.go data/lambda.zip

.PHONY: lambda bindata
