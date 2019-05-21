////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

type Params struct {
	DB
	Groups
	Paths
	Context
	Registration
}

func NewParams() Params {

	params := Params{}

	params.DB.enable = true
	params.Groups.enable = true
	params.Paths.enable = true
	params.Context.enable = true
	params.Registration.enable = true

	return params
}