# elixxir/server

[![pipeline status](https://gitlab.com/elixxir/server/badges/master/pipeline.svg)](https://gitlab.com/elixxir/server/commits/master)
[![coverage report](https://gitlab.com/elixxir/server/badges/master/coverage.svg)](https://gitlab.com/elixxir/server/commits/master)

## Running the Server

First, make sure dependencies are installed into the vendor folder by running
`glide up`. Then, in the project directory, run `go run main.go` with the
appropriate arguments.

If what you're working on requires you to change other repos, you can remove
the other repo from the vendor folder and Go's build tools will look for those
packages in your Go path instead. Knowing which dependencies to remove can be
really helpful if you're changing a lot of repos at once.

If glide isn't working and you don't know why, try removing glide.lock and
~/.glide to brutally cleanse the cache.

Many of these flags override the values set in the config file:

|Long flag|Short flag|Description|Example|
|---|---|---|---|
|--index|-i|Index of the server to start in the list of servers in `server.yaml`|-i 0|
|--batch|-b|Number of messages in a batch (correlated to anonymity set, 1 is the fastest and least anonymous)|-b 64|
|--verbose|-v|Set this to log more messages for debugging|-v|
|--config| |Path to configuration file|--config ~/.elixxir/server.yaml|
|--nodeID|-n|Unique integer identifier for this node. Defaults to be equal to index|-n 125048|
|--profile| |Runs a pprof server at localhost:8087 for profiling. Use to track down unusual and CPU usage.|--profile|
|--version|-V|Print generated version information. To generate, run `$ go generate cmd/version.go`.|--version|
|--help|-h|Print a help message|--help|

Run the `benchmark` subcommand to run the server benchmark: `$ go run main.go benchmark`.

## Config File

Create a directory named `.elixxir` in your home directory with a file 
called `server.yaml` as follows (Make sure to use spaces, not tabs!):

``` yaml
logPath: "server.log"
verbose: "false"
batchSize: 1
dbUsername: "cmix"
dbPassword: ""
dbName: "cmix_server"
dbAddresses:
    - ""
servers:
    - 0.0.0.0:11420
gateways:
    - "0.0.0.0:8443"
# === REQUIRED FOR ENABLING TLS ===
# Path to the server private key file
keyPath: ""
# Path to the server certificate file
certPath: ""
# Path to the gateway certificate file
gatewayCertPath: ""
```

## Project Structure

`benchmark` is for all benchmarks that estimate the performance of the whole 
server. Benchmarks that only test a small subset of the functionality should 
use go test -bench for running and should exist in the package

`cmd` handles command-line flags, configuration options, commands and 
subcommands. This is where the functions that actually start a node are.

`cryptops` contains the code that runs each phase of the mix network. 
Precomputation phases are in `precomputation` and realtime phases are in 
`realtime`.

`globals` contains libraries and variables that many other packages need to 
import, but that don't need to import any packages from `server` itself. In 
general, you shouldn't put things here, and you should redesign things that 
are here so that it makes sense for them to have their own packages.

`io` sets up individual cryptops, phase transitions, and new rounds, and 
handles communication between servers.

`services` contains utilities for the cryptops, including the dispatcher that
allocates cryptop work to different goroutines.

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

