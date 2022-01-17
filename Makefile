
build:
	CGO_ENABLED=0 go build

run: build
	./lyncser sync

test:
	go test -timeout 30s github.com/chrismcgehee/lyncser/sync

mocks:
	mockgen -source=file_store/file_store.go -package=mocks > sync/mocks/mock_file_store.go
	mockgen -source=utils/logger.go -package=mocks > sync/mocks/mock_logger.go
	mockgen -source=utils/reader_encryptor.go -package=mocks > sync/mocks/mock_reader_encryptor.go

docker-build:
	docker build -t lyncser-test --file tests/integration/Dockerfile .

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
