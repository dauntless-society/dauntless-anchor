.PHONY: build clean

# Build for Intel Linux 64-bit
build:
	cd src && go mod tidy && GOOS=linux GOARCH=amd64 go build -o ../dauntless-anchor

clean:
	rm -f dauntless-anchor
