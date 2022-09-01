////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package conf

// Contains Database config params
type Database struct {
	Name     string
	Username string
	Password string
	Address  string
	Port     string
}
