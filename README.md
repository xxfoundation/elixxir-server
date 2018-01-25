privategrity/server
-------------------

[![pipeline status](https://gitlab.com/privategrity/server/badges/master/pipeline.svg)](https://gitlab.com/privategrity/server/commits/master)
[![coverage report](https://gitlab.com/privategrity/server/badges/master/coverage.svg)](https://gitlab.com/privategrity/server/commits/master)

#### CONFIG FILE

Sample config file `sample_server.yaml` located in server directory.
Create a directory named `.privategrity` in your home directory,
move the sample config file into this direcory and rename it to `server.yaml`

Alternatively, here is the text of the config file if you'd like to make it yourself:

``` yaml
logPath: "server.log"
verbose: "false"
servers:
	- 50002
	- 50003
```

#### GODOC GENERATION


- Open terminal and change current directory to your `go/src` directory
- Run the command: `godoc -http=localhost:8000 -goroot=./gitlab.com/`
  - This starts a local webserver with the godocs
- Run the command: `open http://localhost:8000/pkg/`
  - Alternatively open a browser and insert the url manually
   
