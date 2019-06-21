////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

type Permissioning struct {
	Paths            Paths
	Address          string
	RegistrationCode string `yaml:"registrationCode"`
}

func (p Permissioning) GetPublicKey() string {

	regServerPk := "-----BEGIN PUBLIC KEY-----\n" +
		"GuP9Tpgp+0ZEWeBbyjkr7FBnFS+0Olaa08O2i7ythPD/jTHHZ9o+q8/Ahw2Cs5V" +
		"oYQtS8rcrSTu+3m6VLJp/1EqBYeYqkEaCjEpl9AGy8FTr9zduidq1R9ijw9Roke" +
		"eKz8QBVxPL+1sLbKsPjftGuJHzVCBGrOTKuYTV3+9PUtQ0fcflL2p+qFHdoHbw7" +
		"R/vhuxrXCpIBxSZBr+OC/cLMBFH/qiP2VAJ7fvg3o/8GoZOSzokJlthocR6TpMH" +
		"58hPm1WRdltTD1hZ+peyLOm1E4XT0TCIeVsvn9DLWTV/6Tg0YRffKs8rqyLZQt4" +
		"acOjV1i/A6Z2HQqDxbflM46Cruw==\n" +
		"-----END PUBLIC KEY-----\n"

	return regServerPk
}
