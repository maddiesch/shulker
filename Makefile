GOLANG := go

GO_SOURCE := $(shell find . -type f -name "*.go" ! -name "*_test.go")
MIGRATION_FILES := $(shell find ./internal -type f -name "*.sql")

shulker: ${GO_SOURCE} ${MIGRATION_FILES} go.mod go.sum
	${GOLANG} build -o $@ .
