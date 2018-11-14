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
PACKAGES += 'github.com/vapor/mining/tensority/go_algorithm'

BUILD_FLAGS := -ldflags "-X github.com/vapor/version.GitCommit=`git rev-parse HEAD`"

MINER_BINARY32 := miner-$(GOOS)_386
MINER_BINARY64 := miner-$(GOOS)_amd64

BYTOMD_BINARY32 := vapor-$(GOOS)_386
BYTOMD_BINARY64 := vapor-$(GOOS)_amd64

BYTOMCLI_BINARY32 := vaporcli-$(GOOS)_386
BYTOMCLI_BINARY64 := vaporcli-$(GOOS)_amd64

VERSION := $(shell awk -F= '/Version =/ {print $$2}' version/version.go | tr -d "\" ")

MINER_RELEASE32 := miner-$(VERSION)-$(GOOS)_386
MINER_RELEASE64 := miner-$(VERSION)-$(GOOS)_amd64

BYTOMD_RELEASE32 := vapor-$(VERSION)-$(GOOS)_386
BYTOMD_RELEASE64 := vapor-$(VERSION)-$(GOOS)_amd64

BYTOMCLI_RELEASE32 := vaporcli-$(VERSION)-$(GOOS)_386
BYTOMCLI_RELEASE64 := vaporcli-$(VERSION)-$(GOOS)_amd64

BYTOM_RELEASE32 := vapor-$(VERSION)-$(GOOS)_386
BYTOM_RELEASE64 := vapor-$(VERSION)-$(GOOS)_amd64

all: test target release-all

vapor:
	@echo "Building vapor to cmd/vapor/vapor"
	@go build $(BUILD_FLAGS) -o cmd/vapor/vapor cmd/vapor/main.go

vaporcli:
	@echo "Building vaporcli to cmd/vaporcli/vaporcli"
	@go build $(BUILD_FLAGS) -o cmd/vaporcli/vaporcli cmd/vaporcli/main.go

target:
	mkdir -p $@

binary: target/$(BYTOMD_BINARY32) target/$(BYTOMD_BINARY64) target/$(BYTOMCLI_BINARY32) target/$(BYTOMCLI_BINARY64) target/$(MINER_BINARY32) target/$(MINER_BINARY64)

ifeq ($(GOOS),windows)
release: binary
	cd target && cp -f $(MINER_BINARY32) $(MINER_BINARY32).exe
	cd target && cp -f $(BYTOMD_BINARY32) $(BYTOMD_BINARY32).exe
	cd target && cp -f $(BYTOMCLI_BINARY32) $(BYTOMCLI_BINARY32).exe
	cd target && md5sum $(MINER_BINARY32).exe $(BYTOMD_BINARY32).exe $(BYTOMCLI_BINARY32).exe >$(BYTOM_RELEASE32).md5
	cd target && zip $(BYTOM_RELEASE32).zip $(MINER_BINARY32).exe $(BYTOMD_BINARY32).exe $(BYTOMCLI_BINARY32).exe $(BYTOM_RELEASE32).md5
	cd target && rm -f $(MINER_BINARY32) $(BYTOMD_BINARY32) $(BYTOMCLI_BINARY32) $(MINER_BINARY32).exe $(BYTOMD_BINARY32).exe $(BYTOMCLI_BINARY32).exe $(BYTOM_RELEASE32).md5
	cd target && cp -f $(MINER_BINARY64) $(MINER_BINARY64).exe
	cd target && cp -f $(BYTOMD_BINARY64) $(BYTOMD_BINARY64).exe
	cd target && cp -f $(BYTOMCLI_BINARY64) $(BYTOMCLI_BINARY64).exe
	cd target && md5sum $(MINER_BINARY64).exe $(BYTOMD_BINARY64).exe $(BYTOMCLI_BINARY64).exe >$(BYTOM_RELEASE64).md5
	cd target && zip $(BYTOM_RELEASE64).zip $(MINER_BINARY64).exe $(BYTOMD_BINARY64).exe $(BYTOMCLI_BINARY64).exe $(BYTOM_RELEASE64).md5
	cd target && rm -f $(MINER_BINARY64) $(BYTOMD_BINARY64) $(BYTOMCLI_BINARY64) $(MINER_BINARY64).exe $(BYTOMD_BINARY64).exe $(BYTOMCLI_BINARY64).exe $(BYTOM_RELEASE64).md5
else
release: binary
	cd target && md5sum $(MINER_BINARY32) $(BYTOMD_BINARY32) $(BYTOMCLI_BINARY32) >$(BYTOM_RELEASE32).md5
	cd target && tar -czf $(BYTOM_RELEASE32).tgz $(MINER_BINARY32) $(BYTOMD_BINARY32) $(BYTOMCLI_BINARY32) $(BYTOM_RELEASE32).md5
	cd target && rm -f $(MINER_BINARY32) $(BYTOMD_BINARY32) $(BYTOMCLI_BINARY32) $(BYTOM_RELEASE32).md5
	cd target && md5sum $(MINER_BINARY64) $(BYTOMD_BINARY64) $(BYTOMCLI_BINARY64) >$(BYTOM_RELEASE64).md5
	cd target && tar -czf $(BYTOM_RELEASE64).tgz $(MINER_BINARY64) $(BYTOMD_BINARY64) $(BYTOMCLI_BINARY64) $(BYTOM_RELEASE64).md5
	cd target && rm -f $(MINER_BINARY64) $(BYTOMD_BINARY64) $(BYTOMCLI_BINARY64) $(BYTOM_RELEASE64).md5
endif

release-all: clean
	GOOS=darwin  make release
	GOOS=linux   make release
	GOOS=windows make release

clean:
	@echo "Cleaning binaries built..."
	@rm -rf cmd/bytomd/bytomd
	@rm -rf cmd/bytomcli/bytomcli
	@rm -rf cmd/miner/miner
	@rm -rf target
	@echo "Cleaning temp test data..."
	@rm -rf test/pseudo_hsm*
	@rm -rf blockchain/pseudohsm/testdata/pseudo/
	@echo "Cleaning sm2 pem files..."
	@rm -rf crypto/sm2/*.pem
	@echo "Done."

target/$(BYTOMD_BINARY32):
	CGO_ENABLED=0 GOARCH=386 go build $(BUILD_FLAGS) -o $@ cmd/bytomd/main.go

target/$(BYTOMD_BINARY64):
	CGO_ENABLED=0 GOARCH=amd64 go build $(BUILD_FLAGS) -o $@ cmd/bytomd/main.go

target/$(BYTOMCLI_BINARY32):
	CGO_ENABLED=0 GOARCH=386 go build $(BUILD_FLAGS) -o $@ cmd/bytomcli/main.go

target/$(BYTOMCLI_BINARY64):
	CGO_ENABLED=0 GOARCH=amd64 go build $(BUILD_FLAGS) -o $@ cmd/bytomcli/main.go

target/$(MINER_BINARY32):
	CGO_ENABLED=0 GOARCH=386 go build $(BUILD_FLAGS) -o $@ cmd/miner/main.go

target/$(MINER_BINARY64):
	CGO_ENABLED=0 GOARCH=amd64 go build $(BUILD_FLAGS) -o $@ cmd/miner/main.go

test:
	@echo "====> Running go test"
	@go test -tags "network" $(PACKAGES)

benchmark:
	@go test -bench $(PACKAGES)

functional-tests:
	@go test -timeout=5m -tags="functional" ./test 

ci: test functional-tests

.PHONY: all target release-all clean test benchmark
