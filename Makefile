
build:
	CGO_ENABLED=0 go build

run: build
	./lyncser sync --log-level=debug

test:
	go test -timeout 30s github.com/chrismcgehee/lyncser/sync

mocks:
	mockgen -source=file_store/file_store.go -package=mocks > sync/mocks/mock_file_store.go
	mockgen -source=utils/logger.go -package=mocks > sync/mocks/mock_logger.go
	mockgen -source=utils/reader_encryptor.go -package=mocks > sync/mocks/mock_reader_encryptor.go

docker-build:
	docker build -t lyncser-test --file tests/integration/Dockerfile .

new-release:
	@LATEST_RELEASE=$$(git tag | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$$' | sort -V | tail -n 1);\
	NEW_RELEASE=$$(echo $$LATEST_RELEASE | awk -F. -v OFS=. '{$$3++;print}');\
	git tag $$NEW_RELEASE;\
	echo "Creating new release: $$NEW_RELEASE"
	git push --tags

integration-tests: check-env docker-build
	pip3 install pytest
	pytest tests/integration/

check-env:
ifndef GCP_ACCOUNT_CREDENTIALS
	$(error GCP_ACCOUNT_CREDENTIALS is undefined)
endif
ifndef GCP_OAUTH_TOKEN
	$(error GCP_OAUTH_TOKEN is undefined)
endif
