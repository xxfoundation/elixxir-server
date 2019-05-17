////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

type Path interface {
	GetCert() string
	GetKey() string
	GetLog() string
}

type pathImpl struct {
	cert string
	key  string
	log  string
}

func NewPath(cert, key, log string) Path {
	return pathImpl{
		cert: cert,
		key:  key,
		log:  log,
	}
}

func (path pathImpl) GetCert() string {
	return path.cert
}

func (path pathImpl) GetKey() string {
	return path.key
}

func (path pathImpl) GetLog() string {
	return path.log
}
