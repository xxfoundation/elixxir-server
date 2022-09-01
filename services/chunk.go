////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package services

// Chunk are groups of slots that must be completely done before they're ready
// for processing in the next phase of the graph.
// Soon we'll also have a Few type, which will allow a Chunk to be subdivided into
// multiple smaller pieces for processing by multiple threads. This is useful
// for phases like Permute, where the whole thing has to be ready to go before
// transmission is started.
type Chunk struct {
	begin uint32
	end   uint32
}

func NewChunk(begin, end uint32) Chunk {
	return Chunk{begin, end}
}

func (c Chunk) Begin() uint32 {
	return c.begin
}

func (c Chunk) End() uint32 {
	return c.end
}

func (c Chunk) Len() uint32 {
	return c.end - c.begin
}
