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
|--config| |Path to configuration file|--config ~/.elixxir/server.yaml|
|--logLevel|-l|Sets the log message level to print. (0 = info, 1 = debug, >1 = trace)|-l 2|
|--profile| |Runs a pprof server at localhost:8087 for profiling. Use to track down unusual and CPU usage.|--profile|
|--registrationCode| |Required.  Registration code to give to permissioning.| |
|--disableStreaming| |Disables streaming comms. By default true.| |
|--useGPU| |Enables GPU processing| |
|--help|-h|Print a help message|--help|

Run the `benchmark` subcommand to run the server benchmark: `$ go run main.go benchmark`.

## Updating Version Info
```
$ go run main.go generate 
$ mv version_vars.go cmd
```

## Config File

Create a directory named `.elixxir` in your home directory with a file 
called `server.yaml` as follows (Make sure to use spaces, not tabs!):

``` yaml
# START YAML ===
# registration code used for first time registration. Unique. Provided by xx network
registrationCode: "abc123"
useGPU: false
node:
  paths:
    # Path where an error file will be placed in the event of a fatal error
    # used by the wrapper script
    errOutput: ""
    # Path where the ID will be stored after the ID is created on first run
    # used by the wrapper script
    idf:  ""
    # Path to the self signed TLS cert that the node uses for identification
    cert: ""
    # Path to the private key for the self signed TLS cert 
    key:  ""
    # Path to where the log will be stored
    log:  "server.log"
  # port the node will communicate on
  port: 42069
database:
  # information to conenct to the POSTGRESS database storing keys
  name: "node_dbr"
  username: "privacy"
  password: ""
  address: "0.0.0.0:3800"
gateways:
  paths:
    # Path to the self signed TLS cert used by the gateway
    cert: ""
permissioning:
  paths:
    # Path to the self signed TLS cert used by the permissioning. Provided by xx network
    cert: ""
  # IP Address of the permissioning server, provided by xx network
  address: ""
metrics:
  # location of stored metrics data. Modification to set to permissioning
  # server instead of saving will be made at a later date
  log:  "~/.xxnetwork/metrics.log"
# === END YAML
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

