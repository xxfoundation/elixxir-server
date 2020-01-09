package node

import (
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/graphs/precomputation"
	"gitlab.com/elixxir/server/graphs/realtime"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/phase"
	"gitlab.com/elixxir/server/services"
	"time"
)

func NewRoundComponents(gc services.GraphGenerator, topology *connect.Circuit,
	nodeID *id.Node, lastNode *server.LastNode, batchSize uint32, newRoundTimeout int) ([]phase.Phase,
	phase.ResponseMap) {

	responses := make(phase.ResponseMap)

	generalExpectedStates := []phase.State{phase.Active}

	// TODO: Expose this timeout on the command line
	defaultTimeout := time.Duration(newRoundTimeout) * time.Minute

	/*--PRECOMP GENERATE------------------------------------------------------*/

	//Build Precomputation Generation phase and response
	precompGenerateDefinition := phase.Definition{
		Graph:               precomputation.InitGenerateGraph(gc),
		Type:                phase.PrecompGeneration,
		TransmissionHandler: io.TransmitPhase,
		Timeout:             defaultTimeout,
	}
	// On every node but the first, it receives generate and executes generate,
	// First node starts the round via its business logic so it has no
	// receiver for the generate, the first thing in the round
	if !topology.IsFirstNode(nodeID) {
		responses[phase.PrecompGeneration.String()] = phase.NewResponse(
			phase.ResponseDefinition{
				PhaseAtSource:  phase.PrecompGeneration,
				ExpectedStates: generalExpectedStates,
				PhaseToExecute: phase.PrecompGeneration,
			})
	} else {
		//TRANSITION: On first node, generate is received from the last node after
		//every node has completed the phase, it transitions to share phase through
		//a verification state
		responses[phase.PrecompGeneration.String()] =
			phase.NewResponse(
				phase.ResponseDefinition{
					PhaseAtSource:  phase.PrecompShare,
					ExpectedStates: generalExpectedStates,
					PhaseToExecute: phase.PrecompShare,
				})
	}

	/*--PRECOMP SHARE---------------------------------------------------------*/

	// Build Precomputation Share phase and response

	// share needs a copy of the graph constructor with an input size of 1
	gcShare := services.NewGraphGenerator(1, gc.GetErrorHandler(),
		1, 1, 0.0)

	precompShareDefinition := phase.Definition{
		Graph:               precomputation.InitShareGraph(gcShare),
		Type:                phase.PrecompShare,
		TransmissionHandler: io.TransmitPhase,
		Timeout:             defaultTimeout,
		DoVerification:      true,
	}

	// Build response to broadcast of result
	responses[phase.PrecompShare.String()+phase.Verification] =
		phase.NewResponse(
			phase.ResponseDefinition{
				PhaseAtSource:  phase.PrecompShare,
				ExpectedStates: []phase.State{phase.Computed},
				PhaseToExecute: phase.PrecompShare,
			})

	// The last node broadcasts the result to all other nodes so it uses a
	// different transmission handler
	if topology.IsLastNode(nodeID) {
		precompShareDefinition.TransmissionHandler = io.TransmitRoundPublicKey
	}

	// First node transitions into share phase and as a result had no share
	// phase reception
	if !topology.IsFirstNode(nodeID) {
		responses[phase.PrecompShare.String()] = phase.NewResponse(
			phase.ResponseDefinition{
				PhaseAtSource:  phase.PrecompShare,
				ExpectedStates: generalExpectedStates,
				PhaseToExecute: phase.PrecompShare,
			})
	}

	// TRANSITION: the transition out of share phase is done on the first
	// node in the first node check at the bottom of
	// ReceivePostRoundPublicKey in node/receiver.go

	/*--PRECOMP DECRYPT-------------------------------------------------------*/

	// Build Precomputation Decrypt phase and response
	precompDecryptDefinition := phase.Definition{
		Graph:               precomputation.InitDecryptGraph(gc),
		Type:                phase.PrecompDecrypt,
		TransmissionHandler: io.StreamTransmitPhase,
		Timeout:             defaultTimeout,
	}

	// Every node except the first node handles precomp decrypt in the normal
	// pattern

	DecryptResponse := phase.ResponseDefinition{
		PhaseAtSource:  phase.PrecompDecrypt,
		ExpectedStates: generalExpectedStates,
		PhaseToExecute: phase.PrecompDecrypt,
	}

	// TRANSITION: the transition out of decryot phase is done on the first
	// node after every node finishes precomp decrypt and it receives the
	// transmission from the last node.  It transitions into the permute phase
	if topology.IsFirstNode(nodeID) {
		DecryptResponse.PhaseToExecute = phase.PrecompPermute
		DecryptResponse.ExpectedStates = []phase.State{phase.Verified}
	}

	responses[phase.PrecompDecrypt.String()] =
		phase.NewResponse(DecryptResponse)

	/*--PRECOMP PERMUTE-------------------------------------------------------*/

	// Build Precomputation Permute phase and response
	precompPermuteDefinition := phase.Definition{
		Graph:               precomputation.InitPermuteGraph(gc),
		Type:                phase.PrecompPermute,
		TransmissionHandler: io.StreamTransmitPhase,
		Timeout:             defaultTimeout,
	}

	// Every node except the first node handles precomp permute in the normal
	// pattern
	PermuteResponse := phase.ResponseDefinition{
		PhaseAtSource:  phase.PrecompPermute,
		ExpectedStates: generalExpectedStates,
		PhaseToExecute: phase.PrecompPermute,
	}

	// TRANSITION: the transition out of permute phase is done on the first
	// node after every node finishes precomp permute and it receives the
	// transmission from the last node.  It transitions into the reveal phase
	if topology.IsFirstNode(nodeID) {
		PermuteResponse.ExpectedStates = []phase.State{phase.Verified}
		PermuteResponse.PhaseToExecute = phase.PrecompReveal
	}

	responses[phase.PrecompPermute.String()] =
		phase.NewResponse(PermuteResponse)

	/*--PRECOMP REVEAL--------------------------------------------------------*/

	// Build Precomputation Reveal phase and response
	precompRevealDefinition := phase.Definition{
		Graph:               precomputation.InitRevealGraph(gc),
		Type:                phase.PrecompReveal,
		TransmissionHandler: io.StreamTransmitPhase,
		Timeout:             defaultTimeout,
		DoVerification:      true,
	}

	// Every node except the first node handles precomp permute in the normal
	// pattern.  First node has no input because it starts reveal through the
	// transition from permute
	if !topology.IsFirstNode(nodeID) {
		responses[phase.PrecompReveal.String()] = phase.NewResponse(
			phase.ResponseDefinition{
				PhaseAtSource:  phase.PrecompReveal,
				ExpectedStates: generalExpectedStates,
				PhaseToExecute: phase.PrecompReveal})
	}

	// TRANSITION: the transition out of reveal phase is signaled by the last
	// node by broadcasting the PrecompResult message which transfers the result
	// of precomputation to every node and increments the rounds state.
	// The handler ReceivePostPrecompResult in node/receiver.go sends a signal
	// to a control thread on first node which then initiates the realtime.
	// This is tracked through a verification state on all nodes.

	if topology.IsLastNode(nodeID) {
		precompRevealDefinition.TransmissionHandler = io.TransmitPrecompResult
		// Last node also computes the strip operation along with reveal, so its
		// graph is replaced with the composed reveal-strip graph
		precompRevealDefinition.Graph = precomputation.InitStripGraph(gc)
	}

	//All nodes process the verification step
	responses[phase.PrecompReveal.String()+phase.Verification] = phase.NewResponse(
		phase.ResponseDefinition{
			PhaseAtSource:  phase.PrecompReveal,
			ExpectedStates: []phase.State{phase.Computed},
			PhaseToExecute: phase.PrecompReveal})

	/*--REALTIME DECRYPT------------------------------------------------------*/

	// Build Realtime Decrypt phase and response
	realtimeDecryptDefinition := phase.Definition{
		Graph:               realtime.InitDecryptGraph(gc),
		Type:                phase.RealDecrypt,
		TransmissionHandler: io.StreamTransmitPhase,
		Timeout:             defaultTimeout,
	}

	decryptResponse := phase.ResponseDefinition{
		PhaseAtSource:  phase.RealDecrypt,
		ExpectedStates: generalExpectedStates,
		PhaseToExecute: phase.RealDecrypt}

	// TRANSITION: Realtime decrypt is initialized by business logic in
	// server/firstNode.go for the first node, so it has no normal receiver,
	// instead it receives from last node after all nodes have done decrypt
	// and transitions to permute
	if topology.IsFirstNode(nodeID) {
		decryptResponse.ExpectedStates = []phase.State{phase.Verified}
		decryptResponse.PhaseToExecute = phase.RealPermute
	}

	responses[phase.RealDecrypt.String()] = decryptResponse

	/*--REALTIME PERMUTE------------------------------------------------------*/

	// Build Realtime Decrypt phase and response
	realtimePermuteDefinition := phase.Definition{
		Graph:               realtime.InitPermuteGraph(gc),
		Type:                phase.RealPermute,
		TransmissionHandler: io.StreamTransmitPhase,
		Timeout:             defaultTimeout,
		DoVerification:      true,
	}

	//A permute message is never received by first node
	if !topology.IsFirstNode(nodeID) {
		responses[phase.RealPermute.String()] = phase.NewResponse(
			phase.ResponseDefinition{
				PhaseAtSource:  phase.RealPermute,
				ExpectedStates: generalExpectedStates,
				PhaseToExecute: phase.RealPermute})
	}

	//TRANSITION: Last node broadcasts sends the results to the gateways and
	//broadcasts a completed message to all other nodes as a verification step.
	if topology.IsLastNode(nodeID) {
		//build the channel which will be used to send the data
		chanLen := (batchSize + gc.GetOutputSize() - 1) / gc.GetOutputSize()
		chunkChan := make(chan services.Chunk, chanLen)
		//assign the handler
		realtimePermuteDefinition.TransmissionHandler =
			// finish realtime needs access to lastNode to send out the results,
			// an anonymous function is used to wrap the function, passing
			// access while maintaining the transmit signature
			func(network *node.Comms, batchSize uint32,
				roundID id.Round, phaseTy phase.Type, getChunk phase.GetChunk,
				getMessage phase.GetMessage, topology *connect.Circuit,
				nodeID *id.Node, measure phase.Measure) error {
				return io.TransmitFinishRealtime(network, batchSize, roundID,
					phaseTy, getChunk, getMessage, topology, nodeID, lastNode,
					chunkChan, measure)
			}
		//Last node also executes the combined permute-identify graph
		realtimePermuteDefinition.Graph = realtime.InitIdentifyGraph(gc)
	}

	//All nodes process the verification step
	responses[phase.RealPermute.String()+phase.Verification] = phase.NewResponse(
		phase.ResponseDefinition{
			PhaseAtSource:  phase.RealPermute,
			ExpectedStates: []phase.State{phase.Computed},
			PhaseToExecute: phase.RealPermute})

	/*--BUILD PHASES----------------------------------------------------------*/

	//Order here matters, this is the order that phases will be processed in
	phases := []phase.Phase{
		phase.New(precompGenerateDefinition),
		phase.New(precompShareDefinition),
		phase.New(precompDecryptDefinition),
		phase.New(precompPermuteDefinition),
		phase.New(precompRevealDefinition),
		phase.New(realtimeDecryptDefinition),
		phase.New(realtimePermuteDefinition),
	}

	return phases, responses
}
