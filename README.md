# elixxir/server

[![pipeline status](https://gitlab.com/elixxir/server/badges/master/pipeline.svg)](https://gitlab.com/elixxir/server/commits/master)
[![coverage report](https://gitlab.com/elixxir/server/badges/master/coverage.svg)](https://gitlab.com/elixxir/server/commits/master)

## Running the Server

To run the server in CPU development mode:

```
go run main.go --config [configuration-filename]
```

To enable the GPU, you need to install the gpumathsnative library into
`/opt/xxnetwork/lib` (done via make install per its README), then
run with the gpu tag:

```
go run -tags gpu main.go --config [configuration filename]
```

[See gpumaths for more information.](https://gitlab.com/elixxir/gpumaths)

The command line flags for the server can be generated `--help` as follows:

```
$ go run main.go
The server provides a full cMix node for distributed anonymous communications.

Usage:
  server [flags]
  server [command]

Available Commands:
  benchmark   Server benchmarking tests
  generate    Generates version and dependency information for the xx network binary
  help        Help about any command
  version     Print the version and dependency information for the xx network binary

Flags:
  -c, --config string             Path to load the Node configuration file from.
                                  If not set, this file must be named gateway.yaml
                                  and must be located in ~/.xxnetwork/,
                                  /opt/xxnetwork, or /etc/xxnetwork.
      --disableStreaming          Disables streaming comms.
  -h, --help                      help for server
  -l, --logLevel uint             Level of debugging to print (0 = info,
                                  1 = debug, >1 = trace).
      --registrationCode string   Registration code used for first time
                                  registration. This is a unique code provided
                                  by xx network. (Required)
      --useGPU                    Toggles use of the GPU. (default true)

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

If the configuration file is not set via command line flag `--config`, then the
Server configuration file must be named `node.yaml` and be located in one of the
following directories:
1. `$HOME/.xxnetwork/`
2. `/opt/xxnetwork/`
3. `/etc/xxnetwork/`

Server searches for the YAML file in that order and uses the first occurrence
found.

Note: YAML prohibits the use of tabs because whitespace has meaning.

```yaml
# Registration code used for first time registration. This is a unique code
# provided by xx network. (Required)
registrationCode: ""

# Toggles use of the GPU. (Default true)
useGPU: true

# Level of debugging to print (0 = info, 1 = debug, >1 = trace). (Default info)
logLevel: 1

node:
  paths:
    # Path where an error file will be placed in the event of a fatal error.
    # This path is used by the Wrapper Script. (Required)
    errOutput: "/opt/xxnetwork/node-logs/node-err.log"
    # Path to where the identity file (IDF) is saved. The IDF stores the Node's
    # network identity. This is used by the wrapper management script. (Required)
    idf: "/opt/xxnetwork/node-logs/nodeIDF.json"
    # Path to the self-signed TLS certificate for Node. Expects PEM format.
    # (Required)
    cert: "/opt/xxnetwork/creds/node_cert.crt"
    # Path to the private key associated with the self-signed TLS certificate.
    # (Required)
    key: "/opt/xxnetwork/creds/node_key.key"
    # Path where log file will be saved. (Default "./node.log")
    log: "/opt/xxnetwork/node-logs/node.log"
  # Port that the Node will communicate on. (Required)
  port: 11420
  # Local IP address of the Node, used for internal listening. Expects an IPv4
  # address without a port. (default "0.0.0.0")
  listeningAddress: ""
  # The public IPv4 address of the Node, as reported to the network. When not
  # set, external IP address lookup services are used to set this value. If a
  # port is not included, then the port from the port flag is used instead.
  overridePublicIP: ""
  # If set, then it is used to override the internal IP address used to contact
  # yourself. Expects an IPv4 address with or without a port. If no port is
  # included, then the port from the port flag is used.
  overrideInternalIP: ""

# Information to connect to the Postgres database storing keys. (Required)
database:
  name: "cmix_node"
  address: "0.0.0.0:5432"
  username: "cmix"
  password: ""

# Information to communicate with this Node's Gateway.
gateway:
  paths:
    # Path to the self-signed TLS certificate for Gateway. Expects PEM format.
    # (Required)
    cert: "/opt/xxnetwork/creds/gateway_cert.crt"

permissioning:
  paths:
    # Path to the self-signed TLS certificate for the Permissioning server.
    # Expects PEM format. (Required)
    cert: "/opt/xxnetwork/cred/scheduling-cert.crt"
  # Partial IP Address of the network, provided by xx network. (Required)
  address: ""

metrics:
  # Path to store metrics logs.
  log: "/opt/xxnetwork/node-logs/metrics.log"
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

`permissioning` contains logic for dealing with the permissioning server
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

To build with GPU support, add `-tags gpu` after installing gpumathsnative.

### Windows

```
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-w -s' -o server main.go
```

Note: CPU metrics and some time based events may not function in windows.

or

```
GOOS=windows GOARCH=386 CGO_ENABLED=0 go build -ldflags '-w -s' -o server main.go
```

for a 32 bit version.

### Mac OSX

```
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags '-w -s' -o server main.go
```
