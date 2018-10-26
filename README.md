# privategrity/server

[![pipeline status](https://gitlab.com/privategrity/server/badges/master/pipeline.svg)](https://gitlab.com/privategrity/server/commits/master)
[![coverage report](https://gitlab.com/privategrity/server/badges/master/coverage.svg)](https://gitlab.com/privategrity/server/commits/master)

## Running the Server

In project directory, run `$ go run main.go` with optional arguments that will
override the values set in the config file:

|Long flag|Short flag|Description|Example|
|---|---|---|---|
|--index|-i|Index of the server to start in the list of servers in `server.yaml`|-i 0|
|--batch|-b|Number of messages in a batch (correlated to anonymity set, 1 is the fastest and least anonymous)|-b 64|
|--verbose|-v|Set this to log more messages for debugging|-v|
|--config| |Path to configuration file|--config ~/.privategrity/server.yaml|
|--nodeID|-n|Unique integer identifier for this node. Defaults to be equal to index|-n 125048|
|--profile| |Runs a pprof server at localhost:8087 for profiling. Use to track down unusual and CPU usage.|--profile|
|--version|-V|Print generated version information. To generate, run `$ go generate cmd/version.go`.|--version|
|--help|-h|Print a help message|--help|

Run the `benchmark` subcommand to run the server benchmark: `$ go run main.go benchmark`.

## Config File

Create a directory named `.privategrity` in your home directory with a file 
called `server.yaml` as follows:

``` yaml
logPath: "server.log"
verbose: "false"
batchSize: 1
dbUsername: "cmix"
dbPassword: ""
dbName: "cmix_server"
dbAddresses:
    - ""
    - ""
servers:
    - 0.0.0.0:11420
    - 0.0.0.0:11421
```

## Compiling the Binary

To compile a binary that will run the server on your platform,
you will need to run one of the commands in the following sections.
The `.gitlab-ci.yml` file also contains cross build instructions
for all of these platforms.

### Linux

```
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-w -s' -o server main.go
```

### Windows

```
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-w -s' -o server main.go
```

or

```
GOOS=windows GOARCH=386 CGO_ENABLED=0 go build -ldflags '-w -s' -o server main.go
```

for a 32 bit version.

### Mac OSX

```
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-w -s' -o server main.go
```

## Godoc Generation


- Open terminal and change current directory to your `go/src` directory
- Run the command: `godoc -http=localhost:8000 -goroot=./gitlab.com/`
  - This starts a local webserver with the godocs
- Run the command: `open http://localhost:8000/pkg/`
  - Alternatively open a browser and insert the url manually

