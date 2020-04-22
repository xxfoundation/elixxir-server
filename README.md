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
|--logLevel|-l|Sets the log message level to print. (0 = info, 1 = debug, >1 = trace)|-l 2|
|--config| |Path to configuration file|--config ~/.elixxir/server.yaml|
|--nodeID|-n|Unique integer identifier for this node. Defaults to be equal to index|-n 125048|
|--profile| |Runs a pprof server at localhost:8087 for profiling. Use to track down unusual and CPU usage.|--profile|
|--version|-V|Print generated version information. To generate, run `$ go generate cmd/version.go`.|--version|
|--help|-h|Print a help message|--help|
|--metricsWhitespace|-w|Set to print indented metrics JSON files|-w|
|--disableStreaming| |Disables streaming comms. By default true.| |

Run the `benchmark` subcommand to run the server benchmark: `$ go run main.go benchmark`.

## Config File

Create a directory named `.elixxir` in your home directory with a file 
called `server.yaml` as follows (Make sure to use spaces, not tabs!):

``` yaml
# START YAML ===
verbose: true
recoveredErrFile: "/tmp/recovered_error"
logLevel: 1
node:
  id: ""
  paths:
    cert: ""
    key:  ""
    log:  "server.log"
  addresses:
    - "0.0.0.0:11200"
    - "0.0.0.0:11300"
    - "0.0.0.0:11400"
database:
  name: "cmix_server"
  username: "cmix"
  password: ""
  addresses:
    - ""
    - ""
    - ""
gateways:
  paths:
    cert: ""
  addresses:
    - "0.0.0.0:8200"
    - "0.0.0.0:8300"
    - "0.0.0.0:8400"
permissioning:
  paths:
    cert: ""
  address: ""
  registrationCode: ""
  publicKey: "-----BEGIN PUBLIC KEY-----\nMIIDNDCCAiwCggEBAJ22+1lRtmu2/h4UDx0s5VAjdBYf1lON8WSCGGQvC1xIyPek\nGq36GHMkuHZ0+hgisA8ez4E2lD18VXVyZOWhpE/+AS6ZNuAMHT6TELAcfReYBdMF\niyqfS7b5cWv+YRfGtbPMTZvjQRBK1KgK1slOAF9LmT4U8JHrUXQ78zBQw43iNVZ+\nGzTD1qXAzqoaDzaCE8PRmEPQtLCdy5/HLTnI3kHxvxTUu0Vjyig3FiHK0zJLai05\nIUW+v6x0iAUjb1yi/pK4cc2PnDbTKStVCcqMqneirfx7/XfdpvcRJadFb+oVPkMy\nVqImHGoG7TaTeX55lfrVqrvPvj7aJ0HjdUBK4lsCIQDywxGTdM52yTVpkLRlN0oX\n8j+e01CJvZafYcbd6ZmMHwKCAQBcf/awb48UP+gohDNJPkdpxNmIrOW+JaDiSAln\nBxbGE9ewzuaTL4+qfETSyyRSPaU/vk9uw1lYktGqWMQyigbEahVmLn6qcDod7Pi7\nstBdvi65VsFCozhmHRBGHA0TVHIIUFfzSUMJ/6c8YR94syrbtXQMNhyfNb6QmX2y\nAU4u9apheC9Sq+uL1kMsTdCXvFQjsoXa+2DcNk6BYfSio1rKOhCxxNIDzHakcKM6\n/cvdkpWYWavYtW4XJSUteOrGbnG6muPx3SSHGZh0OTzU2DIYaABlR2Dh40wJ5NFV\nF5+ewNxEc/mWvc5u7Ryr7YtvEW962c9QXfD5mONKsnUUsP/nAoIBAEDOApai3bhs\nC3Bq52WT0O23ieyMs5kOTyIi07/ssZy4mifylhjXe7a4tRBsU4xa37n8gfaMFYt8\n9yxfvQ9cgny4VxU0RvqO58NA6QQS5j3utQhzPGwkudJ1OGbiv9VD5fRgClXRAekM\n566lrcJAY8BKGhSShw5ihh0xq6aitENo+UIPM8MO7sfwGXGEqr0KZQCotaebTDkj\nZ8FMz2pCVzO8QPwf6z+VJor6kCXJK2mljHGxOHblMeLW/eri2rB3j92sI1GuB1fP\n/+f9wIa7t61xk1H8dw1ZZkAXIdfNW+R2TktvWBYQHnjIgtzcgN4c39piCyH3Ho7R\nCaz3nlw2QRI=\n-----END PUBLIC KEY-----"
groups:
  cmix:
    prime: "9DB6FB5951B66BB6FE1E140F1D2CE5502374161FD6538DF1648218642F0B5C48C8F7A41AADFA187324B87674FA1822B00F1ECF8136943D7C55757264E5A1A44FFE012E9936E00C1D3E9310B01C7D179805D3058B2A9F4BB6F9716BFE6117C6B5B3CC4D9BE341104AD4A80AD6C94E005F4B993E14F091EB51743BF33050C38DE235567E1B34C3D6A5C0CEAA1A0F368213C3D19843D0B4B09DCB9FC72D39C8DE41F1BF14D4BB4563CA28371621CAD3324B6A2D392145BEBFAC748805236F5CA2FE92B871CD8F9C36D3292B5509CA8CAA77A2ADFC7BFD77DDA6F71125A7456FEA153E433256A2261C6A06ED3693797E7995FAD5AABBCFBE3EDA2741E375404AE25B"
    smallprime: "F2C3119374CE76C9356990B465374A17F23F9ED35089BD969F61C6DDE9998C1F"
    generator: "5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E24809670716C613D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D1AA58C4328A06C46A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A338661D10461C0D135472085057F3494309FFA73C611F78B32ADBB5740C361C9F35BE90997DB2014E2EF5AA61782F52ABEB8BD6432C4DD097BC5423B285DAFB60DC364E8161F4A2A35ACA3A10B1C4D203CC76A470A33AFDCBDD92959859ABD8B56E1725252D78EAC66E71BA9AE3F1DD2487199874393CD4D832186800654760E1E34C09E4D155179F9EC0DC4473F996BDCE6EED1CABED8B6F116F7AD9CF505DF0F998E34AB27514B0FFE7"
  e2e:
    prime: "E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D49413394C049B7A8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688B55B3DD2AEDF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E7861575E745D31F8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC6ADC718DD2A3E041023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C4A530E8FFB1BC51DADDF453B0B2717C2BC6669ED76B4BDD5C9FF558E88F26E5785302BEDBCA23EAC5ACE92096EE8A60642FB61E8F3D24990B8CB12EE448EEF78E184C7242DD161C7738F32BF29A841698978825B4111B4BC3E1E198455095958333D776D8B2BEEED3A1A1A221A6E37E664A64B83981C46FFDDC1A45E3D5211AAF8BFBC072768C4F50D7D7803D2D4F278DE8014A47323631D7E064DE81C0C6BFA43EF0E6998860F1390B5D3FEACAF1696015CB79C3F9C2D93D961120CD0E5F12CBB687EAB045241F96789C38E89D796138E6319BE62E35D87B1048CA28BE389B575E994DCA755471584A09EC723742DC35873847AEF49F66E43873"
    smallprime: "2"
    generator: "2"
metrics:
  log:  "~/.elixxir/metrics.log"
#in ms, omit to wait forever
GatewayConnectionTimeout: 5000 
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

