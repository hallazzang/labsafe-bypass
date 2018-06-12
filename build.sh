echo darwin/amd64
GOOS=darwin GOARCH=amd64 go build -o build/labsafe-darwin.amd64 cmd/labsafe/*.go
echo linux/amd64
GOOS=linux GOARCH=amd64 go build -o build/labsafe-linux.amd64 cmd/labsafe/*.go
echo windows/386
GOOS=windows GOARCH=386 go build -o build/labsafe-windows.386.exe cmd/labsafe/*.go
echo windows/amd64
GOOS=windows GOARCH=amd64 go build -o build/labsafe-windows.amd64.exe cmd/labsafe/*.go

echo zipping
zip -9 build/build.zip build/labsafe-*
