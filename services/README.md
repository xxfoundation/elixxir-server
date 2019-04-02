# Dispatcher

The dispatcher provides an interface for running slot-based
cryptographic operations. These cryptographic operations are run in
the context of a slot inside of a specific round in the cMix system,
as such they can be run in separate processes. The job of the
dispatcher is to manage this parallelism, providing a channel into and
out of the cryptographic operations.

The channels provided by the dispatcher are chained to the network or
message passing interface to communicate between nodes. The dispatcher
also allows the operations to be chained in a single-machine context
for testing and other purposes.

## Basic Operation

The basic usage pattern is:

``` golang
CryptoVar := services.DispatchCryptop(&grp, phase.CryptopName{}, InputChan,
  OutputChan, RoundInfo)
...
// Inside sender thread
data := phase.SlotCryptoName{
  Slot: 12345,
  The: data,
  Goes: here,
}
CryptoVar.InChannel <- data
...
// Receiver thread
result <- CryptoVar.OutChannel
// Do stuff with output slot object
```

This starts a thread that runs the cryptop when slot data is sent to
InputChan and returns results on OutputChan. The types sent to
InputChan and received from OutputChan can be different. Refer to the
cryptop implementation for details.

Each cryptographic operation (cryptop) has 2 inputs, the individual
data in the slot to be processed and the key data stored in the round
structure. Only the slot data needs to be sent over the channel as the
round data is handled during setup. Internally, the cryptops implement
`Build` and `Run` functions to manage their operation. More
information is available from the README in the cryptops folder.

## Chaining Cryptops

Cryptops can be chained as follows:

``` golang
CryptoVar1 := services.DispatchCryptop(&grp, phase.CryptopName1{}, nil,
  nil, RoundInfo)
CryptoVar2 := services.DispatchCryptop(&grp, phase.CryptopName2{},
  CryptoVar1.OutChannel, nil, RoundInfo)
...
// Inside sender thread
data := phase.SlotCryptoName1{
  Slot: 12345,
  The: data,
  Goes: here,
}
CryptoVar1.InChannel <- data
...
// Receiver thread, note we are receiving on the output channel of the
// second Cryptop.
result <- CryptoVar2.OutChannel
// Do stuff with output slot object
```

## Usage in Practice

Except for the LastNode operations (e.g., Peel, refer to cMix
documentation for details), data is typically sent through the nodes
in a predetermined order starting at "Node 1" and flowing through the
system to "Node X", where X is the number of nodes in the system (also
referred to as "LastNode").

In practice, this means that Input channels to the cryptop are fed by
server handler functions, and Output channels are received by and sent
over the network by server sending functions, both defined by the
comms library but implemented in server.

Chains for Cryptops will look like the following:

```
Node 1 Step1 -{NETWORK}-> Node 2 Step 1 -{NETWORK}-> LastNode Step1
  -> LastNode Step 2 -{NETWORK}-> Node 1 Step 3 -{NETWORK}-> LastNode Step 3
```

In this example, Step 2 is a LastNode only operation, which is chained
to LastNode Step1 instead of the network, as is the case in all other
nodes.

To see examples of this, refer to the unit tests in `main_test.go` inside
the server repository.
