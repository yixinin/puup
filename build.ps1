$env:GOOS="windows"
$env:GOARCH=""

go build -o puup.exe ./cmd
go build -o ssh.exe ./cmd/ssh

$env:GOOS="linux"
$env:GOARCH=""
go build -o bin/amd64/linux/puup ./cmd
go build -o bin/amd64/linux/ssh ./cmd/ssh

$env:GOOS="linux"
$env:GOARCH="arm"
go build -o bin/arm/linux/puup ./cmd
go build -o bin/arm/linux/ssh ./cmd/ssh

$env:GOOS="js"
$env:GOARCH="wasm"
go build -o dist/wasm/httpclient.wasm ./cmd/wasm

./deploy.ps1


