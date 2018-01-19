# Cryptops Style Guide
 
When creating a Cryptop, there are 5 main things that need to be created: the Cryptop Structure, a Slot Structure, 
a Key Structure, a `Build` function, and a `Run` function.
The file `precomputation/encrypt.go` has been implemented in accordance with this document as an example.

The CMIX Doc refereed to in this Style Guide can be found [here](https://drive.google.com/open?id=1ha8QtUI9Tk_sCIKWN-QE8YHZ7AKofKrV) 

Cryptop Structure
---

The Cryptop Structure is an empty struct with the name of the Cryptop. 

For example, for the Dispatch Cryptop, the structure will be defined as follows:
 
``` go
type Encrypt struct { }
```
 
Make sure the name is exported by capitalizing the first letter. This is how the Cryptop will be identified. 
Because it is already in a package named either realtime or precomputation, that designation will be used to identify it.
 
Slot Structure
---

A Slot Structure is what is used to send data in and out of the Cryptop. It must comply with the `services.Slot` interface 
in `services/dispatch.go`.  Most Cryptops use the same Slot Structure for incoming data as for outgoing data, but a few 
need different ones.  When using only one Slot Structure, name it:
 
``` go
type SlotCryptop struct {...}
```
 
Where `Cryptop` is the name of the Cryptop.  For example, for the Encrypt Phase, the name would be:
 
``` go
type SlotEncrypt struct {...}
```
 
If the incoming and outgoing data is identical, both input and output can use the same message structure.
If that is not true, two Slots must be used.  In that case, add the words In and Out to the end of the Slot names as follows:

``` golang
type SlotCryptopIn struct {...}

type SlotCryptopOut struct {...}
```
 
#### SLOTID()

The first element in a Slot structure is always a `uint64` called `slot`. 
This cannot be read by the dispatcher, so a function called `SlotID()` must be used to export the data. 
It will just return the slot.  So, for Encrypt, the slot will be implemented as follows:

``` golang
type SlotEncrypt struct {
   	slot uint64
   	...
}
   	
func (e *SlotEncrypt) SlotID() uint64 {
    return (*e).slot
}
```
 
The Slot object must be passed to the `SlotID()` function by reference for efficiency.
The rest of the elements in Slot should be exported and defined by the I/O definition is the CMIX Document for the 
given Phase. Use the name written in the table (in camel case) for all elements except internode keys and their 
encryption keys. For those, use the letters used in the doc.
 
Key Structure
----

The key structure is used to send data stored on the node to the Cryptop.
Only one key structure will be needed per Cryptop. Name it as follows:
 
``` golang
type KeysCryptop struct{...}
```
 
Where `Cryptop` is the name of the Cryptop.  For example, for the Encrypt Phase, the name would be:

``` golang
type KeysEncrypt struct {...}
```
 
The rest of the elements in Keys should be exported and defined by the I/O definition is the CMIX Document.
Use the name written in the table (in camel case) for all elements except internode keys and their encryption keys.
For those use the letters.
 
Build Function
---

The `Build` function is used to create the data structures and allocate memory used when running the Cryptop.
It is implemented as described by the `CryptographicOperation` interface. For Encrypt, its signature is as follows:

``` golang
func (e Encrypt) Build(g *cyclic.Group, face interface{}) *DispatchBuilder {...}
```
 
Inside the function, the first thing you do is get the round object.  That is done as follows:

``` golang
round := face.(*node.Round)
```
 
After that, all that is done is setup.  In general, look at `dispatch_test.go` for specifics.
It must be mentioned is that there is a special way to build arrays of the Slots and Keys.
The arrays need to be made of the interface for the types, and then pointers to the structures need to be stored,
not the structures themselves.  For example, for the slots in encrypt, they are stored as follows:

``` golang
om := make([]services.Slot, round.BatchSize)
 
for i := uint64(0); i < round.BatchSize; i++ {
   	om[i] = &SlotEncrypt{slot: i, ...}
}
```
 
If there are any cryptographic operations that need to be done in build, name the function:
 
  	func buildCryptoCryptop(...) {...}
 
Where `Cryptop` is the name of the Cryptop.  It shouldn't return anything, instead it should modify the data by
passed pointers.  This function should be placed as high up in the `Build` function as possible.
It also must not contain any branching or conditional statements, or any memory allocations.
 
Run Function
---

The `Run` Function is not present in the Cryptop interface because it is called via reflection.
This allows it to be a pure crypto function without and dereferencing or type casing.
The function must be named `Run`, with the first element a pointer to the group object, 
the second a pointer to the input message object, the third a pointer to the output message object, 
and the last a pointer to the key object. It must return the `Message` Interface.
For the Encrypt Cryptop, the signature for the `Run` function is as follows:

``` golang
func (e Encrypt) Run(g * cyclic.Group, in, out *MessageEncrypt, keys *KeysEncrypt) Message {...}
```
 
`Run` should have no conditional or branching statements inside of it.
It is allowed to allocate temp variables, but they should be named `tmp`.
If there are more than one, they should be named `tmp1`, `tmp2`, etc.
Use as few temporary variables as possible. Always use `cyclic.NewMaxInt()` to create new temp
variables. For example:

``` golang
tmp := cyclic.NewMaxInt()
```
 
`Run` must return the `out` object.

Comments
---
Make a comments that describes the phase from a high level perspective immediately above 
the cryptop structure’s definition which starts with the structure’s name.
For the Encrypt structure it would look like this:

``` golang
// Implements the realtime Encrypt Phase. In this phase the Second Unpermuted 
// Internode Message Keys are applies as well as the Reception Keys  
type Encrypt struct {}
```

For every line of code inside of `Run` and `buildCrypto`, place a line of comment immediately above the line of code.
That line should describe in mathematical terms what is being done and reference and equation number which it is implementing.

File Structure
---

When writing your cryptop, put the elements in the file in the following order:

``` golang
type Cryptop struct {}
type SlotCryptopIn struct {...}
func (e * SlotCryptopIn) SlotID() uint64 {...}
type SlotCryptopOut struct {...}
func (e * SlotCryptopOut) Slot() uint64 {...}
type KeysCryptop struct {...}
func (e Cryptop) Build(g *cyclic.Group, face interface{}) *DispatchBuilder {...}
func (e Cryptop) Run(g * cyclic.Group, in *SlotCryptopIn, 
out *SlotCryptopOut, keys *KeysCryptop ) Slot {...}
func buildCryptoCryptop(...) {...}
```
