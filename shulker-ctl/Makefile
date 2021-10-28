GO_FILES := $(shell find . -type f -name '*.go' ! -name '*_test.go')

# All the go files
GOLANG ?= go

shulker-ctl: ${GO_FILES}
	${GOLANG} build -o shulker-ctl .

.PHONY: release
release:
	@mkdir -p ../release
	GOOS=linux GOARCH=amd64 ${GOLANG} build -o ../release/shulker-ctl-Linux-x86_64
	GOOS=linux GOARCH=arm64 ${GOLANG} build -o ../release/shulker-ctl-Linux-arm64
	GOOS=darwin GOARCH=amd64 ${GOLANG} build -o ../release/shulker-ctl-Darwin-x86_64
	GOOS=darwin GOARCH=arm64 ${GOLANG} build -o ../release/shulker-ctl-Darwin-arm64

TEST_REGEX ?= .

.PHONY: test
test:
	go test -count=1 -v -run=${TEST_REGEX} ./...
