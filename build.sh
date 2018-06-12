echo darwin/amd64
GOOS=darwin GOARCH=amd64 go build -o build/labsafe-darwin.amd64 cmd/labsafe/*.go
echo linux/amd64
GOOS=linux GOARCH=amd64 go build -o build/labsafe-linux.amd64 cmd/labsafe/*.go
echo windows/386
GOOS=windows GOARCH=386 go build -o build/labsafe-windows.386.exe cmd/labsafe/*.go
echo windows/amd64
GOOS=windows GOARCH=amd64 go build -o build/labsafe-windows.amd64.exe cmd/labsafe/*.go

echo compressing
GZIP=-9 tar -C build -cvzf build/labsafe-darwin.amd64.tar.gz labsafe-darwin.amd64
GZIP=-9 tar -C build -cvzf build/labsafe-linux.amd64.tar.gz labsafe-linux.amd64
cd build
zip -9 -FS labsafe-windows.386.zip labsafe-windows.386.exe
zip -9 -FS labsafe-windows.amd64.zip labsafe-windows.amd64.exe
