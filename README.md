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
database:
    username: "cmix"
    password: ""
    name: "cmix_server"
    addresses:
        - ""
groups:
    cmix:
        prime: >-
            0x
            9DB6FB5951B66BB6FE1E140F1D2CE5502374161FD6538DF1648218642F0B5C48
            C8F7A41AADFA187324B87674FA1822B00F1ECF8136943D7C55757264E5A1A44F
            FE012E9936E00C1D3E9310B01C7D179805D3058B2A9F4BB6F9716BFE6117C6B5
            B3CC4D9BE341104AD4A80AD6C94E005F4B993E14F091EB51743BF33050C38DE2
            35567E1B34C3D6A5C0CEAA1A0F368213C3D19843D0B4B09DCB9FC72D39C8DE41
            F1BF14D4BB4563CA28371621CAD3324B6A2D392145BEBFAC748805236F5CA2FE
            92B871CD8F9C36D3292B5509CA8CAA77A2ADFC7BFD77DDA6F71125A7456FEA15
            3E433256A2261C6A06ED3693797E7995FAD5AABBCFBE3EDA2741E375404AE25B
        smallprime: >-
            0x
            F2C3119374CE76C9356990B465374A17F23F9ED35089BD969F61C6DDE9998C1F
        generator: >-
            0x
            5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E24809670716C613
            D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D1AA58C4328A06C4
            6A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A338661D10461C0D135472
            085057F3494309FFA73C611F78B32ADBB5740C361C9F35BE90997DB2014E2EF5
            AA61782F52ABEB8BD6432C4DD097BC5423B285DAFB60DC364E8161F4A2A35ACA
            3A10B1C4D203CC76A470A33AFDCBDD92959859ABD8B56E1725252D78EAC66E71
            BA9AE3F1DD2487199874393CD4D832186800654760E1E34C09E4D155179F9EC0
            DC4473F996BDCE6EED1CABED8B6F116F7AD9CF505DF0F998E34AB27514B0FFE7
    e2e:
        prime: >-
            0x
            9DB6FB5951B66BB6FE1E140F1D2CE5502374161FD6538DF1648218642F0B5C48
            C8F7A41AADFA187324B87674FA1822B00F1ECF8136943D7C55757264E5A1A44F
            FE012E9936E00C1D3E9310B01C7D179805D3058B2A9F4BB6F9716BFE6117C6B5
            B3CC4D9BE341104AD4A80AD6C94E005F4B993E14F091EB51743BF33050C38DE2
            35567E1B34C3D6A5C0CEAA1A0F368213C3D19843D0B4B09DCB9FC72D39C8DE41
            F1BF14D4BB4563CA28371621CAD3324B6A2D392145BEBFAC748805236F5CA2FE
            92B871CD8F9C36D3292B5509CA8CAA77A2ADFC7BFD77DDA6F71125A7456FEA15
            3E433256A2261C6A06ED3693797E7995FAD5AABBCFBE3EDA2741E375404AE25B
        smallprime: >-
            0x
            F2C3119374CE76C9356990B465374A17F23F9ED35089BD969F61C6DDE9998C1F
        generator: >-
            0x
            5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E24809670716C613
            D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D1AA58C4328A06C4
            6A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A338661D10461C0D135472
            085057F3494309FFA73C611F78B32ADBB5740C361C9F35BE90997DB2014E2EF5
            AA61782F52ABEB8BD6432C4DD097BC5423B285DAFB60DC364E8161F4A2A35ACA
            3A10B1C4D203CC76A470A33AFDCBDD92959859ABD8B56E1725252D78EAC66E71
            BA9AE3F1DD2487199874393CD4D832186800654760E1E34C09E4D155179F9EC0
            DC4473F996BDCE6EED1CABED8B6F116F7AD9CF505DF0F998E34AB27514B0FFE7
verbose: "false"
batchSize: 4
servers:
    - localhost:50000
    - localhost:50001
    - localhost:50002
    - localhost:50003
    - localhost:50004
gateways:
    - "localhost:8440"
# === REQUIRED FOR ENABLING TLS ===
# Path to the server cert/key/log & gateway cert
paths:
    cert: "../keys/cmix.rip.crt"
    key:  "../keys/cmix.rip.key"
    log:  "server.log"
    gatewayCert: "../keys/gateway.cmix.rip.crt"
# Skip registration server check when registering users
skipReg: "true"

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

