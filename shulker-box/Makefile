GO_FILES := $(shell find . -type f -name '*.go' ! -name '*_test.go')

# All the go files
GOLANG ?= go

shulker-box: ${GO_FILES}
	${GOLANG} build -o shulker-box .

.PHONY: release
release:
	@mkdir -p ../release
	GOOS=linux GOARCH=amd64 ${GOLANG} build -o ../release/shulker-box-Linux-x86_64
	GOOS=linux GOARCH=arm64 ${GOLANG} build -o ../release/shulker-box-Linux-arm64
	GOOS=darwin GOARCH=amd64 ${GOLANG} build -o ../release/shulker-box-Darwin-x86_64
	GOOS=darwin GOARCH=arm64 ${GOLANG} build -o ../release/shulker-box-Darwin-arm64

TEST_REGEX ?= .

.PHONY: test
test:
	go test -count=1 -v -run=${TEST_REGEX} ./...
