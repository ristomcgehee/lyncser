The entrypoint for the application is [main.go](main.go). It uses a library called [cobra](https://github.com/spf13/cobra)
to read command line arguments. For logging, it uses a library called [zap](https://github.com/uber-go/zap).

## sync package
The main logic for the application is in [sync/sync.go](sync/sync.go). This is where the decisions are made for whether a
file needs to be synced or not. Whenever it makes calls to upload or download files, it utilizes interfaces such as 
[FileStore](sync/file_store.go). This is primarily to make unit testing feasible. It has the side benefit of making it
easier to add new remote file stores in the future.

There are unit tests in this package covering the logic for syncing as well for deleting remote files. This occurs in
[sync/sync_test.go](sync/sync_test.go). The test cases themselves are written in [Gherkin](https://cucumber.io/docs/gherkin/reference/)
format in the [sync/features](sync/features) directory. These tests are run using a Behavior Driven Development (BDD) framework
called [go-bdd](https://github.com/go-bdd/gobdd). To run the tests, run:
```sh
make test
```
These unit tests use mock implementations of several interfaces in order to run quickly and to avoid the complications of
testing dependencies such as Google Drive. When you make changes to the interfaces used in this package, you make need to
re-generate the mock implementations. To do this, run:
```sh
make mocks
```

## file_store package
The file_store package contains the interfaces and implementations for uploading and downloading files. Interacting with
the local file system should be done using [files_store/local_file_store.go](files_store/local_file_store.go). This is so
that unit tests do not need to actually touch the file system.

## tests/integration folder
Integration tests are tests that test the application from end-to-end using a real Google Drive account. Because Go is a
fairly low-level language, these tests are written in Python. In these tests, multiple clients are created, each in their
own Docker container running the application. These tests require two environment variables to be set: LYNCSER_CREDENTIALS
and LYNCSER_TOKEN. The former is the contents of a Google Cloud Platform service account credentials file. The latter is
the OAuth token generated when running lyncser and performing the authorization step (what is written to 
~/config/lyncser/token.json). If both those variables are set, you can run with:
```sh
make integration-tests
```

## vendor folder
The vendor folder contains the Go dependencies for the application. This means that when `go build` is run, it does not
need to download any dependencies from the internet. The reason for doing this is to speed up Docker builds.
