# privategrity/server

[![pipeline status](https://gitlab.com/privategrity/server/badges/master/pipeline.svg)](https://gitlab.com/privategrity/server/commits/master)
[![coverage report](https://gitlab.com/privategrity/server/badges/master/coverage.svg)](https://gitlab.com/privategrity/server/commits/master)

## Running the Command Line Client

In project directory, run `go run main.go` with optional arguments that will
override the values set in the config file:

`-i <INT>`: Index of the server to start in the list of servers in `server.yaml`

`-b <INT>`: Batch size for the server

`-v`      : Boolean indicating verbose logging

## Config File

Sample config file `sample_server.yaml` located in server directory.
Create a directory named `.privategrity` in your home directory,
move the sample config file into this direcory and rename it to `server.yaml`

Alternatively, here is the text of the config file if you'd like to
make it yourself:

``` yaml
logPath: "server.log"
verbose: "false"
batchSize: 1
servers:
	- 50002
	- 50003
```

## Compiling the Binary

To compile a binary that will run the server on your platform,
you will need to run one of the commands in the following sections.
The `gitlab-ci.yml` file also contains cross build instructions
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

## GODOC GENERATION


- Open terminal and change current directory to your `go/src` directory
- Run the command: `godoc -http=localhost:8000 -goroot=./gitlab.com/`
  - This starts a local webserver with the godocs
- Run the command: `open http://localhost:8000/pkg/`
  - Alternatively open a browser and insert the url manually

