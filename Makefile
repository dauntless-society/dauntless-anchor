.PHONY: build clean deploy

# Build for Intel Linux 64-bit
build:
	cd src && go mod tidy && GOOS=linux GOARCH=amd64 go build -o ../bin/dauntless-anchor

deploy: build
	@test -n "$$DAUNTLESS_API_HOST" || (echo "DAUNTLESS_API_HOST is required" >&2; exit 1)
	scp bin/dauntless-anchor root@$${DAUNTLESS_API_HOST}:/var/lib/dauntless/bin/dauntless-anchor.canidate

clean:
	rm -f bin/dauntless-anchor
