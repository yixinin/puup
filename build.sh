go build -o puup ./cmd
go build -o ssh ./cmd/ssh

export GOOS=linux
export GOARCH=amd64
go build -o bin/amd64/linux/puup ./cmd
go build -o bin/amd64/linux/ssh ./cmd/ssh

export GOOS=linux
export GOARCH=arm64
go build -o bin/arm64/linux/puup ./cmd
go build -o bin/arm64/linux/ssh ./cmd/ssh

export GOOS=linux
export GOARCH=arm
go build -o bin/arm/linux/puup ./cmd
go build -o bin/arm/linux/ssh ./cmd/ssh

export GOOS="js"
export GOARCH="wasm"
go build -o dist/wasm/httpclient.wasm ./cmd/wasm