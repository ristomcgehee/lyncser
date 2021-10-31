
build:
	go build

run: build
	./lyncser

test:
	go test -timeout 30s github.com/chrismcgehee/lyncser/sync

mocks:
	mockgen -source=utils/file_store.go -package=sync > sync/mock_file_store.go

docker-build:
	docker build -t lyncser .

update-bashrc: build
	docker run \
		-v $(CURDIR):/lyncser \
		-v $(shell realpath ~/.config/lyncser/globalConfig.yaml):/root/.config/lyncser/globalConfig.yaml \
		-v $(shell realpath ~/.config/lyncser/token.json):/root/.config/lyncser/token.json \
		-v $(shell realpath ~/.config/lyncser/credentials.json):/root/.config/lyncser/credentials.json \
		lyncser \
		bash -c "/lyncser/lyncser && \
			echo '# YO!!!' >> /root/.bashrc && \
			/lyncser/lyncser"

no-global: build
	docker run \
		-v $(CURDIR):/lyncser \
		-v $(shell realpath ~/.config/lyncser/token.json):/root/.config/lyncser/token.json \
		-v $(shell realpath ~/.config/lyncser/credentials.json):/root/.config/lyncser/credentials.json \
		lyncser \
		bash -c "/lyncser/lyncser && \
			/lyncser/lyncser"

docker-interactive: build
	docker run -it \
		-v $(CURDIR):/lyncser \
		-v $(shell realpath ~/.config/lyncser/globalConfig.yaml):/root/.config/lyncser/globalConfig.yaml \
		-v $(shell realpath ~/.config/lyncser/token.json):/root/.config/lyncser/token.json \
		-v $(shell realpath ~/.config/lyncser/credentials.json):/root/.config/lyncser/credentials.json \
		lyncser 

RELEASE_DIR := $(shell mktemp -d)
release: build
	rm -f lyncser-amd64.tar.gz
	mkdir $(RELEASE_DIR)/lyncser
	cp lyncser install/* $(RELEASE_DIR)/lyncser
	tar czf lyncser-amd64.tar.gz --directory=$(RELEASE_DIR) .
