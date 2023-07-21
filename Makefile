GOLANG := go

GO_SOURCE := $(shell find . -type f -name "*.go" ! -name "*_test.go")

shulker: ${GO_SOURCE} go.mod go.sum
	${GOLANG} build -o $@ .
