privategrity/server
-------------------

[![pipeline status](https://gitlab.com/privategrity/server/badges/master/pipeline.svg)](https://gitlab.com/privategrity/server/commits/master)
[![coverage report](https://gitlab.com/privategrity/server/badges/master/coverage.svg)](https://gitlab.com/privategrity/server/commits/master)

CONFIG FILE
Sample config file "sample_server.yaml" located in server directory. Create a directory named ".privategrity" in your home directory, move the sample config file into this direcory and rename it to "server.yaml"

Alternatively, here is the text of the config file if you'd like to make it yourself:
logPath: "server.log"
verbose: "false"
servers:
	- 50002
	- 50003
