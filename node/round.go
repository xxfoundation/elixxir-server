///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package node

import (
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/gpumaths"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/graphs/precomputation"
	"gitlab.com/elixxir/server/graphs/realtime"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/services"
	"time"
)

// round.go creates the components for a round

// NewRoundComponents sets up the transitions of different phases in the round
func NewRoundComponents(gc services.GraphGenerator, topology *connect.Circuit,
	nodeID *id.ID, instance *internal.Instance, batchSize uint32,
	newRoundTimeout time.Duration, pool *gpumaths.StreamPool,
	disableStreaming bool) (
	[]phase.Phase, phase.ResponseMap) {

	responses := make(phase.ResponseMap)

	generalExpectedStates := []phase.State{phase.Active}

	// Used to swap between streaming and non-streaming
	transmissionHandler := io.StreamTransmitPhase

	/*--PRECOMP GENERATE------------------------------------------------------*/

	//Build Precomputation Generation phase and response
	precompGenerateDefinition := phase.Definition{
		Graph:               precomputation.InitGenerateGraph(gc),
		Type:                phase.PrecompGeneration,
		TransmissionHandler: io.TransmitPhase,
		Timeout:             newRoundTimeout,
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
	gcShare := services.NewGraphGenerator(1,
		1, 1, 0.0)

	precompShareDefinition := phase.Definition{
		Graph:               precomputation.InitShareGraph(gcShare),
		Type:                phase.PrecompShare,
		TransmissionHandler: io.TransmitPhase,
		Timeout:             newRoundTimeout,
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

	// Swap the transmission handler if using streaming
	if disableStreaming {
		transmissionHandler = io.TransmitPhase
	}

	// Build Precomputation Decrypt phase and response
	precompDecryptDefinition := phase.Definition{
		Graph:               precomputation.InitDecryptGraph(gc),
		Type:                phase.PrecompDecrypt,
		TransmissionHandler: transmissionHandler,
		Timeout:             newRoundTimeout,
	}
	if pool != nil {
		precompDecryptDefinition.Graph = precomputation.InitDecryptGPUGraph(gc)
	} else {
		precompDecryptDefinition.Graph = precomputation.InitDecryptGraph(gc)
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
	// Swap the transmission handler if using streaming
	if disableStreaming {
		transmissionHandler = io.TransmitPhase
	}

	// Build Precomputation Permute phase and response
	precompPermuteDefinition := phase.Definition{
		Graph:               precomputation.InitPermuteGraph(gc),
		Type:                phase.PrecompPermute,
		TransmissionHandler: transmissionHandler,
		Timeout:             newRoundTimeout,
	}
	if pool != nil {
		precompPermuteDefinition.Graph = precomputation.InitPermuteGPUGraph(gc)
	} else {
		precompPermuteDefinition.Graph = precomputation.InitPermuteGraph(gc)
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

	// Swap the transmission handler if using streaming
	if disableStreaming {
		transmissionHandler = io.TransmitPhase
	}

	// Build Precomputation Reveal phase and response
	precompRevealDefinition := phase.Definition{
		Type:                phase.PrecompReveal,
		TransmissionHandler: transmissionHandler,
		Timeout:             newRoundTimeout,
		DoVerification:      true,
	}
	if pool != nil {
		precompRevealDefinition.Graph = precomputation.InitRevealGPUGraph(gc)
	} else {
		precompRevealDefinition.Graph = precomputation.InitRevealGraph(gc)
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
		if pool != nil {
			precompRevealDefinition.Graph = precomputation.InitStripGPUGraph(gc)
		} else {
			precompRevealDefinition.Graph = precomputation.InitStripGraph(gc)
		}
	}

	//All nodes process the verification step
	responses[phase.PrecompReveal.String()+phase.Verification] = phase.NewResponse(
		phase.ResponseDefinition{
			PhaseAtSource:  phase.PrecompReveal,
			ExpectedStates: []phase.State{phase.Computed},
			PhaseToExecute: phase.PrecompReveal})

	/*--REALTIME DECRYPT------------------------------------------------------*/
	// Swap the transmission handler if using streaming
	if disableStreaming {
		transmissionHandler = io.TransmitPhase
	}

	// Build Realtime Decrypt phase and response
	realtimeDecryptDefinition := phase.Definition{
		Graph:               realtime.InitDecryptGraph(gc),
		Type:                phase.RealDecrypt,
		TransmissionHandler: transmissionHandler,
		Timeout:             newRoundTimeout,
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
	// Swap the transmission handler if using streaming
	if disableStreaming {
		transmissionHandler = io.TransmitPhase
	}

	// Build Realtime Decrypt phase and response
	realtimePermuteDefinition := phase.Definition{
		Graph:               realtime.InitPermuteGraph(gc),
		Type:                phase.RealPermute,
		TransmissionHandler: transmissionHandler,
		Timeout:             newRoundTimeout,
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
		//assign the handler
		realtimePermuteDefinition.TransmissionHandler =
			// finish realtime needs access to lastNode to send out the results,
			// an anonymous function is used to wrap the function, passing
			// access while maintaining the transmit signature
			func(roundID id.Round, instance phase.GenericInstance, getChunk phase.GetChunk, getMessage phase.GetMessage) error {
				return io.TransmitFinishRealtime(roundID, instance, getChunk, getMessage)
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
