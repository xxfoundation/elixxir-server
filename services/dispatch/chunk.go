package dispatch

// Lots are groups of slots that must be completely done before they're ready
// for processing in the next phase of the graph.
// Soon we'll also have a Few type, which will allow a Lot to be subdivided into
// multiple smaller pieces for processing by multiple threads. This is useful
// for phases like Permute, where the whole thing has to be ready to go before
// transmission is started.
type Lot struct {
	begin uint32
	end   uint32
}

func NewLot(begin, end uint32) Lot {
	return Lot{begin, end}
}

func (c Lot) Begin() uint32 {
	return c.begin
}

func (c Lot) End() uint32 {
	return c.end
}

func (c Lot) Len() uint32 {
	return c.end - c.begin
}
