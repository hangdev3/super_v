.PHONY: build package package-test test

VERSION ?= $(if $(SUPERV_VERSION),$(SUPERV_VERSION),0.1.0)

build:
	go build -buildvcs=false -o build/super-v ./cmd/super-v

package:
	SUPERV_VERSION=$(VERSION) bash ./packaging/build-deb.sh

package-test: package
	bash ./packaging/test-deb.sh build/super-v_$(VERSION)_$(shell dpkg --print-architecture).deb

test:
	go test -count=1 ./...
