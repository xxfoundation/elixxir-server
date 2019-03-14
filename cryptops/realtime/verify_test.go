////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package realtime

import (
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/verification"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
	"testing"
)

func TestRealTimeVerify(t *testing.T) {
	var im []services.Slot
	batchSize := uint64(2)
	round := globals.NewRound(batchSize)

	grp := cyclic.NewGroup(cyclic.NewIntFromString(
		"FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1"+
			"29024E088A67CC74020BBEA63B139B22514A08798E3404DD"+
			"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245"+
			"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED"+
			"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D"+
			"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F"+
			"83655D23DCA3AD961C62F356208552BB9ED529077096966D"+
			"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B"+
			"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9"+
			"DE2BCBF6955817183995497CEA956AE515D2261898FA0510"+
			"15728E5A8AAAC42DAD33170D04507A33A85521ABDF1CBA64"+
			"ECFB850458DBEF0A8AEA71575D060C7DB3970F85A6E1E4C7"+
			"ABF5AE8CDB0933D71E8C94E04A25619DCEE3D2261AD2EE6B"+
			"F12FFA06D98A0864D87602733EC86A64521F2B18177B200C"+
			"BBE117577A615D6C770988C0BAD946E208E24FA074E5AB31"+
			"43DB5BFCE0FD108E4B82D120A92108011A723C12A787E6D7"+
			"88719A10BDBA5B2699C327186AF4E23C1A946834B6150BDA"+
			"2583E9CA2AD44CE8DBBBC2DB04DE8EF92E8EFC141FBECAA6"+
			"287C59474E6BC05D99B2964FA090C3A2233BA186515BE7ED"+
			"1F612970CEE2D7AFB81BDD762170481CD0069127D5B05AA9"+
			"93B4EA988D8FDDC186FFB7DC90A6C08F4DF435C934063199"+
			"FFFFFFFFFFFFFFFF", 16), cyclic.NewInt(5),
		cyclic.NewInt(23),
		cyclic.NewRandom(cyclic.NewInt(1), cyclic.NewInt(42)))

	user := id.NewUserFromUint(42, t)
	assocData := format.NewAssociatedData()
	assocData.SetRecipient(user)

	csprig := csprng.NewSystemRNG()

	data := make([]byte, format.AD_KEYFP_LEN)
	csprig.Read(data)
	fp := format.NewFingerprint(data)
	assocData.SetKeyFingerprint(*fp)

	data = make([]byte, format.AD_TIMESTAMP_LEN)
	csprig.Read(data)
	assocData.SetTimestamp(data)

	data = make([]byte, format.AD_MAC_LEN)
	csprig.Read(data)
	assocData.SetMAC(data)

	*fp = assocData.GetKeyFingerprint()
	payloadMicList := [][]byte{
		assocData.GetRecipientID(),
		fp[:],
		assocData.GetTimestamp(),
		assocData.GetMAC(),
	}
	copy(assocData.GetRecipientMIC(), verification.GenerateMIC(payloadMicList, uint64(format.AD_RMIC_LEN)))

	im = append(im, &Slot{
		Slot:           0,
		AssociatedData: cyclic.NewIntFromBytes(assocData.SerializeAssociatedData())})

	im = append(im, &Slot{
		Slot:           1,
		AssociatedData: cyclic.NewIntFromBytes(assocData.SerializeAssociatedData())})

	im = append(im, &Slot{
		Slot:           2,
		AssociatedData: cyclic.NewInt(0)})

	ExpectedOutputs := []bool{true, true, false}

	dc := services.DispatchCryptop(&grp, Verify{}, nil, nil, round)

	for i := uint64(0); i < batchSize; i++ {
		dc.InChannel <- &im[i]
		trn := <-dc.OutChannel

		rtn := (*trn).(*Slot)
		ExpectedOutput := ExpectedOutputs[i]

		if round.MIC_Verification[rtn.Slot] != ExpectedOutput {
			t.Errorf("%v - Expected: %v, Got: %v", i, round.MIC_Verification[rtn.Slot],
				ExpectedOutput)
		}
	}
}

// Smoke test test the identify function
func TestVerifyRun(t *testing.T) {
	keys := KeysVerify{
		Verification: new(bool)}

	grp := cyclic.NewGroup(cyclic.NewIntFromString(
		"FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1"+
			"29024E088A67CC74020BBEA63B139B22514A08798E3404DD"+
			"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245"+
			"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED"+
			"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D"+
			"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F"+
			"83655D23DCA3AD961C62F356208552BB9ED529077096966D"+
			"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B"+
			"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9"+
			"DE2BCBF6955817183995497CEA956AE515D2261898FA0510"+
			"15728E5A8AAAC42DAD33170D04507A33A85521ABDF1CBA64"+
			"ECFB850458DBEF0A8AEA71575D060C7DB3970F85A6E1E4C7"+
			"ABF5AE8CDB0933D71E8C94E04A25619DCEE3D2261AD2EE6B"+
			"F12FFA06D98A0864D87602733EC86A64521F2B18177B200C"+
			"BBE117577A615D6C770988C0BAD946E208E24FA074E5AB31"+
			"43DB5BFCE0FD108E4B82D120A92108011A723C12A787E6D7"+
			"88719A10BDBA5B2699C327186AF4E23C1A946834B6150BDA"+
			"2583E9CA2AD44CE8DBBBC2DB04DE8EF92E8EFC141FBECAA6"+
			"287C59474E6BC05D99B2964FA090C3A2233BA186515BE7ED"+
			"1F612970CEE2D7AFB81BDD762170481CD0069127D5B05AA9"+
			"93B4EA988D8FDDC186FFB7DC90A6C08F4DF435C934063199"+
			"FFFFFFFFFFFFFFFF", 16), cyclic.NewInt(5),
		cyclic.NewInt(23),
		cyclic.NewRandom(cyclic.NewInt(1), cyclic.NewInt(42)))

	user := id.NewUserFromUint(42, t)
	assocData := format.NewAssociatedData()
	assocData.SetRecipient(user)

	csprig := csprng.NewSystemRNG()

	data := make([]byte, format.AD_KEYFP_LEN)
	csprig.Read(data)
	csprig.Read(data)
	fp := format.NewFingerprint(data)
	assocData.SetKeyFingerprint(*fp)

	data = make([]byte, format.AD_TIMESTAMP_LEN)
	csprig.Read(data)
	assocData.SetTimestamp(data)

	data = make([]byte, format.AD_MAC_LEN)
	csprig.Read(data)
	assocData.SetMAC(data)

	payloadMicList := [][]byte{
		assocData.GetRecipientID(),
		fp[:],
		assocData.GetTimestamp(),
		assocData.GetMAC(),
	}
	copy(assocData.GetRecipientMIC(), verification.GenerateMIC(payloadMicList, uint64(format.AD_RMIC_LEN)))

	im := Slot{
		Slot:           0,
		AssociatedData: cyclic.NewIntFromBytes(assocData.SerializeAssociatedData())}

	om := Slot{
		Slot:           0,
		AssociatedData: cyclic.NewInt(0)}

	verify := Verify{}
	verify.Run(&grp, &im, &om, &keys)

	if !*keys.Verification {
		t.Errorf("Expected: %v, Got: %v", true,
			*keys.Verification)
	}

	im = Slot{
		Slot:           0,
		AssociatedData: cyclic.NewInt(0)}

	om = Slot{
		Slot:           0,
		AssociatedData: cyclic.NewInt(0)}

	verify.Run(&grp, &im, &om, &keys)

	if *keys.Verification {
		t.Errorf("Expected: %v, Got: %v", false,
			*keys.Verification)
	}
}
