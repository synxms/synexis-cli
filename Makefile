build_new:
	GOOS=darwin GOARCH=arm64 go build -o bin/synexis-darwin-arm64
	GOOS=darwin GOARCH=amd64 go build -o bin/synexis-darwin-amd64
	GOOS=windows GOARCH=amd64 go build -o bin/synexis-windows-amd64.exe
	GOOS=linux GOARCH=amd64 go build -o bin/synexis-linux-amd64
	GOOS=freebsd GOARCH=amd64 go build -o bin/synexis-freebsd-amd64
