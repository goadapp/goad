all: osx linux windows

lambda:
	GOOS=linux GOARCH=amd64 go build -o data/lambda/goad-lambda ./lambda
	zip -jr data/lambda data/lambda

bindata: lambda
	go-bindata -nocompress -pkg infrastructure -o infrastructure/bindata.go data/lambda.zip

linux: bindata
	GOOS=linux GOARCH=amd64 go build -o ./build/linux/x86-64/goad ./cli
	GOOS=linux GOARCH=386 go build -o ./build/linux/x86/goad ./cli

osx: bindata
	GOOS=darwin GOARCH=amd64 go build -o ./build/osx/x86-64/goad ./cli

windows: bindata
	GOOS=windows GOARCH=amd64 go build -o ./build/windows/x86-64/goad ./cli
	GOOS=windows GOARCH=386 go build -o ./build/windows/x86/goad ./cli

clean:
	rm -rf data/lambda/goad-lambda
	rm -rf build

all-zip: all
	mkdir ./build/zip
	zip -jr ./build/zip/goad-osx-x86-64 ./build/osx/x86-64/goad
	zip -jr ./build/zip/goad-linux-x86-64 ./build/linux/x86-64/goad
	zip -jr ./build/zip/goad-linux-x86 ./build/linux/x86/goad
	zip -jr ./build/zip/goad-windows-x86-64 ./build/windows/x86-64/goad
	zip -jr ./build/zip/goad-windows-x86 ./build/windows/x86/goad

.PHONY: lambda bindata
