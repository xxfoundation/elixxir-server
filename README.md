# elixxir/server

[![pipeline status](https://gitlab.com/elixxir/server/badges/master/pipeline.svg)](https://gitlab.com/elixxir/server/commits/master)
[![coverage report](https://gitlab.com/elixxir/server/badges/master/coverage.svg)](https://gitlab.com/elixxir/server/commits/master)

## Running the Server

To run the server in cpu development mode:

```
go run main.go --config [configuration-filename]
```

To enable the GPU, you need to install the gpumaths library into
`/opt/xxnetwork/lib` (done via make install per it's README), then
run with the gpu tag:

```
go run -tags gpu main.go --config [configuration filename]
```

[See gpumaths for more information.](https://gitlab.com/elixxir/gpumaths)

The command line flags for the server can be generated `--help` as follows:

```
$ go run main.go
Error: required flag(s) "config" not set
Usage:
  server [flags]
  server [command]

Available Commands:
  benchmark   Server benchmarking tests
  generate    Generates version and dependency information for the Elixxir binary
  help        Help about any command
  version     Print the version and dependency information for the Elixxir binary

Flags:
      --config string             Required.  config file (default is $HOME/.elixxir/server.yaml)
      --disableStreaming          Disables streaming comms.
  -h, --help                      help for server
  -l, --logLevel uint             Level of debugging to display. 0 = info, 1 = debug, >1 = trace (default 1)
      --registrationCode string   Required.  Registration code to give to permissioning
      --useGPU                    Toggle on GPU

Use "server [command] --help" for more information about a command.
```

All of those flags, except `--config`, override values in the configuration
file.

The `version` subcommand prints the version:

```
$ go run main.go version
Elixxir Server v1.1.0 -- fac1a93d fix version cmd

Dependencies:

module gitlab.com/elixxir/server
...
```

The `benchmark` subcommand is currently unsupported, but you can run (CPU)
server benchmarks with it:

```
$ go run main.go benchmark
```

The `generate` subcommand is used for updating version information (see the
next section).

## Updating Version Info
```
$ go run main.go generate
$ mv version_vars.go cmd
```

## Config File

Create a directory named `.xxnetwork` in your home directory with a file
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

`benchmark` is for all benchmarks that estimate the performance of the
whole server. Benchmarks that only test a small subset of the
functionality should use go test -bench for running and should exist
in the package. It is currently limited to CPU-only benchmarks.

`cmd` handles command-line flags, configuration options, commands and
subcommands. This is where the functions that actually start a node
are.

`cryptops` contains the code that runs each phase of the mix network.
Precomputation phases are in `precomputation` and realtime phases are
in `realtime`.

`globals` contains libraries and variables that many other packages
need to import, but that don't need to import any packages from
`server` itself. In general, you shouldn't put things here, and you
should redesign things that are here so that it makes sense for them
to have their own packages.

`internal` contains internal server data structures.

`io` sets up individual cryptops, phase transitions, and new rounds,
and handles communication between servers.

`services` contains utilities for the cryptops, including the
dispatcher that allocates cryptop work to different goroutines.

`node` contains node business logic.

`permissioning` contins logic for dealing with the permissioning server
(the current source of consensus).

## Compiling the Binary

To compile a binary that will run the server on your platform,
you will need to run one of the commands in the following sections.
The `.gitlab-ci.yml` file also contains cross build instructions
for all of these platforms.

Note: GPU support is only provided on Linux.

### Linux

```
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-w -s' -o server main.go
```

To build with GPU support, add `-tags gpu` after installing gpumaths.

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
