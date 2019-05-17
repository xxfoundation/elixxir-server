////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

type DB interface {
	GetDBName() string
	GetAddresses() []string
	GetUsername() string
	GetPassword() string
}

type dbImpl struct {
	addresses []string
	username  string
	password  string // TODO: maybe this should be a secure string via memguard
	dbName    string
}

func NewDB(addresses []string, username, password, dbName string) DB {
	return dbImpl{
		addresses: addresses,
		username:  username,
		password:  password,
		dbName:    dbName,
	}
}

func (db dbImpl) GetDBName() string {
	return db.dbName
}

func (db dbImpl) GetAddresses() []string {
	return db.addresses
}

func (db dbImpl) GetUsername() string {
	return db.username
}

func (db dbImpl) GetPassword() string {
	return db.password
}
