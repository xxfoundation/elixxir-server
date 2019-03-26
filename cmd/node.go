////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package node contains the initialization and main loop of a cMix server.
package cmd

import (
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/cryptops/realtime"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/io"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// RunRealtime controls when realtime is kicked off and which
// messages are sent through the realtime phase. It reads up to batchSize
// messages from the MessageCh, then reads a round and kicks off realtime
// with those messages.
func RunRealTime(batchSize uint64, MessageCh chan *realtime.Slot,
	RoundCh chan *string, realtimeSignal *sync.Cond) {
	msgCount := uint64(0)
	msgList := make([]*realtime.Slot, batchSize)
	for msg := range MessageCh {
		jww.DEBUG.Printf("Adding message ("+
			"%d/%d) from SenderID %q with Associated Data %s...",
			msgCount+1, batchSize, msg.CurrentID,
			msg.AssociatedData.Text(10))
		msg.Slot = msgCount
		msgList[msgCount] = msg
		msgCount = msgCount + 1

		if msgCount == batchSize {
			msgCount = uint64(0)
			// Pass the batch queue into Realtime and begin

			roundID := *(<-RoundCh)
			// Keep reading until we get a valid round
			for globals.GlobalRoundMap.GetRound(roundID) == nil {
				roundID = *(<-RoundCh)
			}

			startTime := time.Now()
			jww.INFO.Printf("Realtime phase with Round ID %s started at %s\n",
				roundID, startTime.Format(time.RFC3339))
			io.KickoffDecryptHandler(roundID, batchSize, msgList)

			// Signal the precomputation thread to run
			realtimeSignal.L.Lock()
			realtimeSignal.Signal()
			realtimeSignal.L.Unlock()

			// Wait for the realtime phase to complete and record the elapsed time
			go func(roundID string, startTime time.Time) {
				// Since we just started, it is safe not to check for nil here.
				// this is still technically a race cond, though
				round := globals.GlobalRoundMap.GetRound(roundID)
				round.WaitUntilPhase(globals.REAL_COMPLETE)
				endTime := time.Now()
				jww.INFO.Printf("Realtime phase with Round ID %s finished at %s!\n",
					roundID, endTime.Format(time.RFC3339))
				jww.INFO.Printf("Realtime phase completed in %d ms",
					int64(endTime.Sub(startTime)/time.Millisecond))
				globals.GlobalRoundMap.DeleteRound(roundID)
			}(roundID, startTime)
		}
	}
}

// RunPrecomputation controls when precomputation is kicked off. It monitors
// the length of the RoundCh and creates new rounds and kicks of precomputation
// whenever it falls below a threshold.

// Number of currently executing precomputations
var numRunning = int32(0)

// Maximum number of simultaneously run precomputation
var numPrecompSimultaneous int

// Size of the buffer for input messages
var messageBufferSize int

// Maximum number of stored precomputations
const PRECOMP_BUFFER_SIZE = int(100)

func RunPrecomputation(RoundCh chan *string, realtimeSignal *sync.Cond) {

	var timer *time.Timer

	realtimeChan := make(chan bool, PRECOMP_BUFFER_SIZE+1)
	precompChan := make(chan bool, PRECOMP_BUFFER_SIZE+1)

	go readSignal(realtimeChan, realtimeSignal)

	for {

		timer = time.NewTimer(333 * time.Millisecond)

		select {
		case <-realtimeChan:
		case <-precompChan:
		case <-timer.C:
		}

		timer.Stop()

		for checkPrecompBuffer(len(RoundCh), int(atomic.LoadInt32(&numRunning))) {
			// Begin the round on all nodes
			startTime := time.Now()
			roundID := globals.PeekNextRoundID()

			jww.INFO.Printf("Precomputation phase with Round ID %s started at %s\n",
				roundID, startTime.Format(time.RFC3339))
			atomic.AddInt32(&numRunning, int32(1))
			io.BeginNewRound(io.Servers, roundID)
			// Wait for round to be in the PRECOMP_COMPLETE state before
			// adding it to the round map
			go func(RoundCh chan *string, precompChan chan bool,
				roundID string, startTime time.Time) {

				round := globals.GlobalRoundMap.GetRound(roundID)

				// Wait until the round completes to continue
				round.WaitUntilPhase(globals.PRECOMP_COMPLETE)
				atomic.AddInt32(&numRunning, int32(-1))
				if round.GetPhase() == globals.ERROR {
					jww.ERROR.Printf("Error occurred during precomputation"+
						" of round %s, round aborted", roundID)
				} else {
					endTime := time.Now()
					jww.INFO.Printf("Precomputation phase with Round ID %s finished at %s!\n",
						roundID, endTime.Format(time.RFC3339))
					jww.INFO.Printf("Precomputation phase completed in %d ms",
						int64(endTime.Sub(startTime)/time.Millisecond))

					RoundCh <- &roundID
					precompChan <- true
				}

			}(RoundCh, precompChan, roundID, startTime)
		}
	}
}

func checkPrecompBuffer(numRounds, numRunning int) bool {
	return (numRounds+numRunning < PRECOMP_BUFFER_SIZE) && (numRunning < numPrecompSimultaneous)
}

func readSignal(rDone chan bool, realtimeSignal *sync.Cond) {
	for true {
		realtimeSignal.L.Lock()
		realtimeSignal.Wait()
		realtimeSignal.L.Unlock()
		rDone <- true
	}

}

// StartServer reads configuration options and starts the cMix server
func StartServer(serverIndex int, batchSize uint64) {
	viper.Debug()
	jww.INFO.Printf("Log Filename: %v\n", viper.GetString("logPath"))
	jww.INFO.Printf("Config Filename: %v\n", viper.ConfigFileUsed())

	//Set the max number of processes
	runtime.GOMAXPROCS(runtime.NumCPU() * 2)

	//Start the performance monitor
	go MonitorMemoryUsage()

	// Set global batch size
	globals.BatchSize = batchSize
	jww.INFO.Printf("Batch Size: %v\n", globals.BatchSize)

	gateways := viper.GetStringSlice("gateways")
	if len(gateways) < 1 {
		// No gateways in config file or passed via command line
		jww.FATAL.Panicf("Error: No gateways specified! Add to" +
			" configuration file!")
		return
	} else {
		// List of gateways found in config file, select one to use
		// TODO: For now, just use the first one?
		globals.GatewayAddress = gateways[0]
	}

	// Initialize the backend
	dbAddresses := viper.GetStringSlice("dbAddresses")
	dbAddress := ""
	if (serverIndex >= 0) && (serverIndex < len(dbAddresses)) {
		// There's a DB address for this server in the list and we can use it
		dbAddress = dbAddresses[serverIndex]
	}
	globals.Users = globals.NewUserRegistry(
		viper.GetString("dbUsername"),
		viper.GetString("dbPassword"),
		viper.GetString("dbName"),
		dbAddress,
	)
	globals.PopulateDummyUsers(globals.GetGroup())

	// Get all servers
	io.Servers = getServers(serverIndex)

	serverList := viper.GetStringSlice("servers")[0]
	for i := 1; i < len(viper.GetStringSlice("servers")); i++ {
		serverList = serverList + "," + viper.GetStringSlice("servers")[i]
	}
	jww.INFO.Print("Server list: " + serverList)

	// TODO Generate globals.Grp somewhere intelligent
	primeString := "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
		"29024E088A67CC74020BBEA63B139B22514A08798E3404DD" +
		"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245" +
		"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED" +
		"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D" +
		"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F" +
		"83655D23DCA3AD961C62F356208552BB9ED529077096966D" +
		"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B" +
		"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9" +
		"DE2BCBF6955817183995497CEA956AE515D2261898FA0510" +
		"15728E5A8AACAA68FFFFFFFFFFFFFFFF"

	prime := large.NewInt(0)
	prime.SetString(primeString, 16)
	// one := cyclic.NewInt(1)
	grp := cyclic.NewGroup(prime, large.NewInt(2), large.NewInt(4))

	globals.SetGroup(&grp)

	// Start mix servers on localServer
	localServer := io.Servers[serverIndex]
	jww.INFO.Printf("Starting server on %v\n", localServer)
	// Initialize GlobalRoundMap and waiting rounds queue
	globals.GlobalRoundMap = globals.NewRoundMap()

	// ensure that the Node ID is populated
	viperNodeID := uint64(viper.GetInt("nodeid"))
	if viperNodeID == 0 {
		id.SetNodeID(uint64(serverIndex))
	} else {
		id.SetNodeID(viperNodeID)
	}

	certPath := viper.GetString("certPath")
	keyPath := viper.GetString("keyPath")
	gatewayCertPath := viper.GetString("gatewayCertPath")
	// Set the certPaths explicitly to avoid data races
	connect.ServerCertPath = certPath
	connect.GatewayCertPath = gatewayCertPath
	// Kick off Comms server
	go node.StartServer(localServer, io.NewServerImplementation(), certPath,
		keyPath)

	// TODO Replace these concepts with a better system
	id.IsLastNode = serverIndex == len(io.Servers)-1
	io.NextServer = io.Servers[(serverIndex+1)%len(io.Servers)]

	// Block until we can reach every server
	io.VerifyServersOnline()

	globals.RoundRecycle = make(chan *globals.Round, PRECOMP_BUFFER_SIZE)

	// Run as many as half the number of nodes times the number of
	// passthroughs (which is 4).
	numPrecompSimultaneous = int((uint64(len(io.Servers)) * 4) / 2)

	messageBufferSize = int(10 * batchSize)

	if messageBufferSize < 1000 {
		messageBufferSize = 1000
	}

	if id.IsLastNode {
		realtimeSignal := &sync.Cond{L: &sync.Mutex{}}
		io.RoundCh = make(chan *string, PRECOMP_BUFFER_SIZE)
		io.MessageCh = make(chan *realtime.Slot, messageBufferSize)
		// Last Node handles when realtime and precomp get run
		go RunRealTime(batchSize, io.MessageCh, io.RoundCh, realtimeSignal)
		go RunPrecomputation(io.RoundCh, realtimeSignal)
	}

	// Main loop
	run()
}

// Main server loop
func run() {
	io.TimeUp = time.Now().UnixNano()
	// Run a roundtrip ping every couple seconds if last node
	if id.IsLastNode {
		ticker := time.NewTicker(5 * time.Second)
		quit := make(chan struct{})
		go func() {
			for {
				select {
				case <-ticker.C:
					jww.DEBUG.Print("Starting Roundtrip Ping")
					io.GetRoundtripPing(io.Servers)
					io.GetServerMetrics(io.Servers)
				case <-quit:
					ticker.Stop()
					return
				}
			}
		}()
		// Blocks forever as a keepalive
		select {}
	} else {
		// Blocks forever as a keepalive
		select {}
	}
	jww.ERROR.Printf("Node Exiting without error")
}

// getServers pulls a string slice of server ports from the config file and
// verifies that the ports are valid.
func getServers(serverIndex int) []string {
	servers := viper.GetStringSlice("servers")
	if servers == nil {
		jww.FATAL.Panicf("No servers listed in config file!")
	}
	for i := range servers {
		// Split address and port
		s := strings.Split(servers[i], ":")
		// Convert port to an int
		temp, err := strconv.Atoi(s[1])
		// catch non-int ports
		if err != nil {
			jww.FATAL.Panicf("Non-integer server ports in config file!")
		}
		// Catch invalid ports
		if temp > 65535 || temp < 0 {
			jww.FATAL.Panicf("Port %v listed in the config file is not a "+
				"valid port!", temp)
		}
		// Catch reserved ports
		if temp < 1024 {
			jww.WARN.Printf("Port %v is a reserved port, superuser privilege"+
				" may be required!", temp)
		}
		if i == serverIndex {
			// Remove the IP from the local server
			// in order to listen on the relevant port
			servers[i] = "0.0.0.0:" + s[1]
		}
	}
	return servers
}
