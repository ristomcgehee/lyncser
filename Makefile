
build:
	CGO_ENABLED=0 go build

run: build
	./lyncser sync

test:
	go test -timeout 30s github.com/chrismcgehee/lyncser/sync

mocks:
	mockgen -source=utils/file_store.go -package=sync > sync/mock_file_store.go
	mockgen -source=utils/logger.go -package=sync > sync/mock_logger.go

docker-build:
	docker build -t lyncser-test .


new-tag:
	# Update the tag number manually
	git tag v0.1.11
	git push --tags

integration-tests: check-env docker-build
	pip3 install pytest
	pytest tests/integration/

check-env:
ifndef LYNCSER_CREDENTIALS
	$(error LYNCSER_CREDENTIALS is undefined)
endif
ifndef LYNCSER_TOKEN
	$(error LYNCSER_TOKEN is undefined)
endif
