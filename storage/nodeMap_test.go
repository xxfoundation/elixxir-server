///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package storage

import (
	"git.xx.network/xx_network/primitives/id"
	"testing"
	"time"
)

// Hidden function for one-time unit testing database implementation
// DROP TABLE clients;
//func TestDatabaseImpl(t *testing.T) {
//	jwalterweatherman.SetLogThreshold(jwalterweatherman.LevelTrace)
//	jwalterweatherman.SetStdoutThreshold(jwalterweatherman.LevelTrace)
//
//	db, err := newDatabase("cmix", "", "cmix_server", "0.0.0.0", "5432", false)
//	if err != nil {
//		t.Errorf(err.Error())
//		return
//	}
//
//	testId := id.NewIdFromString("test", id.User, t)
//
//	err = db.UpsertClient(&Client{
//		Id:             testId.Marshal(),
//		DhKey:        make([]byte, 0),
//		PublicKey:      make([]byte, 0),
//		Nonce:          make([]byte, 0),
//		NonceTimestamp: time.Now(),
//		IsRegistered:   false,
//	})
//	if err != nil {
//		t.Errorf(err.Error())
//		return
//	}
//	err = db.UpsertClient(&Client{
//		Id:             testId.Marshal(),
//		DhKey:        testId.Marshal(),
//		PublicKey:      testId.Marshal(),
//		Nonce:          testId.Marshal(),
//		NonceTimestamp: time.Now(),
//		IsRegistered:   true,
//	})
//	if err != nil {
//		t.Errorf(err.Error())
//		return
//	}
//
//	client, err := db.GetClient(testId)
//	if err != nil {
//		t.Errorf(err.Error())
//		return
//	}
//	jwalterweatherman.INFO.Printf("Obtained client %+v", client)
//}

// Happy path
func TestMapImpl_GetClient(t *testing.T) {
	testId := id.NewIdFromString("test", id.User, t)
	m := &MapImpl{
		clients: make(map[id.ID]*Client),
	}
	m.clients[*testId] = &Client{Id: testId.Marshal(), IsRegistered: true}
	result, err := m.GetClient(testId)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	if !result.IsRegistered {
		t.Errorf("Did not get expected client")
	}
}

// Error path
func TestMapImpl_GetClientError(t *testing.T) {
	testId := id.NewIdFromString("test", id.User, t)
	missingId := id.NewIdFromString("Zezima", id.User, t)
	m := &MapImpl{
		clients: make(map[id.ID]*Client),
	}
	m.clients[*testId] = &Client{Id: testId.Marshal(), IsRegistered: true}
	result, err := m.GetClient(missingId)
	if err == nil {
		t.Errorf("Expected error, returned a result: %+v", result)
		return
	}
}

// Happy path
func TestMapImpl_UpsertClient(t *testing.T) {
	testId := id.NewIdFromString("test", id.User, t)
	m := &MapImpl{
		clients: make(map[id.ID]*Client),
	}

	testClient := &Client{
		Id:             testId.Marshal(),
		DhKey:          nil,
		PublicKey:      nil,
		Nonce:          nil,
		NonceTimestamp: time.Now(),
		IsRegistered:   false,
	}

	err := m.UpsertClient(testClient)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	if _, ok := m.clients[*testId]; !ok {
		t.Errorf("Failed to insert client")
		return
	}

	newClient := &Client{
		Id:             testId.Marshal(),
		DhKey:          testId.Marshal(),
		PublicKey:      testId.Marshal(),
		Nonce:          testId.Marshal(),
		NonceTimestamp: time.Now().Add(1 * time.Second),
		IsRegistered:   true,
	}
	err = m.UpsertClient(newClient)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	result := m.clients[*testId]
	if !result.IsRegistered {
		t.Errorf("Expected client to be updated, got: %+v", result)
	}
}
