
build:
	CGO_ENABLED=0 go build

run: build
	./lyncser sync --log-level=debug

test:
	go test -timeout 30s github.com/ristomcgehee/lyncser/sync

mocks:
	mockgen -source=filestore/file_store.go -package=mocks > sync/mocks/mock_file_store.go
	mockgen -source=utils/logger.go -package=mocks > sync/mocks/mock_logger.go
	mockgen -source=utils/reader_encryptor.go -package=mocks > sync/mocks/mock_reader_encryptor.go

docker-build:
	docker build -t lyncser-test --file tests/integration/Dockerfile .

new-release:
	@LATEST_RELEASE=$$(git tag | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$$' | sort -V | tail -n 1);\
	NEW_RELEASE=$$(echo $$LATEST_RELEASE | awk -F. -v OFS=. '{$$3++;print}');\
	sed -i "s/const appVersion = \"v[0-9]*\.[0-9]*\.[0-9]*\"/const appVersion = \"$$NEW_RELEASE\"/g" main.go;\
	git add main.go;\
	git commit -m "Bump version to $$NEW_RELEASE";\
	git push;\
	git tag $$NEW_RELEASE;\
	echo "Creating new release: $$NEW_RELEASE";\
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

.PHONY: install
install: ## Installs all dependencies needed
	@echo Installing tools from tools/tools.go
	cd tools; cat tools.go | grep _ | awk -F'"' '{print $$2}' | xargs -tI % go install %

lint:
	golangci-lint run -c .golangci.yml
