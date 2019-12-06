ifndef GOOS
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	GOOS := darwin
else ifeq ($(UNAME_S),Linux)
	GOOS := linux
else
$(error "$$GOOS is not defined. If you are using Windows, try to re-make using 'GOOS=windows make ...' ")
endif
endif

PACKAGES    := $(shell go list ./... | grep -v '/vendor/' | grep -v '/crypto/ed25519/chainkd' | grep -v '/mining/tensority')

BUILD_FLAGS := -ldflags "-X github.com/bytom/vapor/version.GitCommit=`git rev-parse HEAD`"


VAPORD_BINARY32 := vapord-$(GOOS)_386
VAPORD_BINARY64 := vapord-$(GOOS)_amd64

VAPORCLI_BINARY32 := vaporcli-$(GOOS)_386
VAPORCLI_BINARY64 := vaporcli-$(GOOS)_amd64

VERSION := $(shell awk -F= '/Version =/ {print $$2}' version/version.go | tr -d "\" ")


VAPORD_RELEASE32 := vapord-$(VERSION)-$(GOOS)_386
VAPORD_RELEASE64 := vapord-$(VERSION)-$(GOOS)_amd64

VAPORCLI_RELEASE32 := vaporcli-$(VERSION)-$(GOOS)_386
VAPORCLI_RELEASE64 := vaporcli-$(VERSION)-$(GOOS)_amd64

VAPOR_RELEASE32 := vapor-$(VERSION)-$(GOOS)_386
VAPOR_RELEASE64 := vapor-$(VERSION)-$(GOOS)_amd64

all: test target release-all install

fedd:
	@echo "Building fedd to cmd/fedd/fedd"
	@go build $(BUILD_FLAGS) -o cmd/fedd/fedd cmd/fedd/main.go

precognitive:
	@echo "Building precognitive to cmd/precognitive/precognitive"
	@go build $(BUILD_FLAGS) -o cmd/precognitive/precognitive cmd/precognitive/main.go

vapord:
	@echo "Building vapord to cmd/vapord/vapord"
	@go build $(BUILD_FLAGS) -o cmd/vapord/vapord cmd/vapord/main.go

vaporcli:
	@echo "Building vaporcli to cmd/vaporcli/vaporcli"
	@go build $(BUILD_FLAGS) -o cmd/vaporcli/vaporcli cmd/vaporcli/main.go

install:
	@echo "Installing vapord and vaporcli to $(GOPATH)/bin"
	@go install ./cmd/vapord
	@go install ./cmd/vaporcli

target:
	mkdir -p $@

binary: target/$(VAPORD_BINARY32) target/$(VAPORD_BINARY64) target/$(VAPORCLI_BINARY32) target/$(VAPORCLI_BINARY64)

ifeq ($(GOOS),windows)
release: binary
	cd target && cp -f $(VAPORD_BINARY32) $(VAPORD_BINARY32).exe
	cd target && cp -f $(VAPORCLI_BINARY32) $(VAPORCLI_BINARY32).exe
	cd target && md5sum $(VAPORD_BINARY32).exe $(VAPORCLI_BINARY32).exe >$(VAPOR_RELEASE32).md5
	cd target && zip $(VAPOR_RELEASE32).zip $(VAPORD_BINARY32).exe $(VAPORCLI_BINARY32).exe $(VAPOR_RELEASE32).md5
	cd target && rm -f $(VAPORD_BINARY32) $(VAPORCLI_BINARY32) $(VAPORD_BINARY32).exe $(VAPORCLI_BINARY32).exe $(VAPOR_RELEASE32).md5
	cd target && cp -f $(VAPORD_BINARY64) $(VAPORD_BINARY64).exe
	cd target && cp -f $(VAPORCLI_BINARY64) $(VAPORCLI_BINARY64).exe
	cd target && md5sum $(VAPORD_BINARY64).exe $(VAPORCLI_BINARY64).exe >$(VAPOR_RELEASE64).md5
	cd target && zip $(VAPOR_RELEASE64).zip $(VAPORD_BINARY64).exe $(VAPORCLI_BINARY64).exe $(VAPOR_RELEASE64).md5
	cd target && rm -f $(VAPORD_BINARY64) $(VAPORCLI_BINARY64) $(VAPORD_BINARY64).exe $(VAPORCLI_BINARY64).exe $(VAPOR_RELEASE64).md5
else
release: binary
	cd target && md5sum $(VAPORD_BINARY32) $(VAPORCLI_BINARY32) >$(VAPOR_RELEASE32).md5
	cd target && tar -czf $(VAPOR_RELEASE32).tgz $(VAPORD_BINARY32) $(VAPORCLI_BINARY32) $(VAPOR_RELEASE32).md5
	cd target && rm -f $(VAPORD_BINARY32) $(VAPORCLI_BINARY32) $(VAPOR_RELEASE32).md5
	cd target && md5sum $(VAPORD_BINARY64) $(VAPORCLI_BINARY64) >$(VAPOR_RELEASE64).md5
	cd target && tar -czf $(VAPOR_RELEASE64).tgz $(VAPORD_BINARY64) $(VAPORCLI_BINARY64) $(VAPOR_RELEASE64).md5
	cd target && rm -f $(VAPORD_BINARY64) $(VAPORCLI_BINARY64) $(VAPOR_RELEASE64).md5
endif

release-all: clean
	GOOS=darwin  make release
	GOOS=linux   make release
	GOOS=windows make release

clean:
	@echo "Cleaning binaries built..."
	@rm -rf cmd/vapord/vapord
	@rm -rf cmd/vaporcli/vaporcli
	@rm -rf target
	@rm -rf $(GOPATH)/bin/vapord
	@rm -rf $(GOPATH)/bin/vaporcli
	@echo "Cleaning temp test data..."
	@rm -rf test/pseudo_hsm*
	@rm -rf blockchain/pseudohsm/testdata/pseudo/
	@echo "Cleaning sm2 pem files..."
	@rm -rf crypto/sm2/*.pem
	@echo "Done."

target/$(VAPORD_BINARY32):
	CGO_ENABLED=0 GOARCH=386 go build $(BUILD_FLAGS) -o $@ cmd/vapord/main.go

target/$(VAPORD_BINARY64):
	CGO_ENABLED=0 GOARCH=amd64 go build $(BUILD_FLAGS) -o $@ cmd/vapord/main.go

target/$(VAPORCLI_BINARY32):
	CGO_ENABLED=0 GOARCH=386 go build $(BUILD_FLAGS) -o $@ cmd/vaporcli/main.go

target/$(VAPORCLI_BINARY64):
	CGO_ENABLED=0 GOARCH=amd64 go build $(BUILD_FLAGS) -o $@ cmd/vaporcli/main.go


test:
	@echo "====> Running go test"
	@go test -tags "network" $(PACKAGES)

benchmark:
	@go test -bench $(PACKAGES)

functional-tests:
	@go test -timeout=5m -tags="functional" ./test 

ci: test

.PHONY: all target release-all clean test benchmark
