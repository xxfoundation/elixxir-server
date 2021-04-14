///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Handles the Map backend for node storage

package storage

import (
	"github.com/pkg/errors"
	"gitlab.com/xx_network/primitives/id"
)

// GetClient returns a Client from Map with the given ID
// Or an error if a matching Client does not exist
func (m *MapImpl) GetClient(id *id.ID) (*Client, error) {
	m.Lock()
	defer m.Unlock()

	if val, ok := m.clients[*id]; ok {
		return val, nil
	} else {
		return nil, errors.Errorf("Unable to locate Client for ID %s", id.String())
	}
}

// UpsertClient inserts the given Client into Map if it does not exist
// Or updates the Map Client if its value does not match the given Client
func (m *MapImpl) UpsertClient(client *Client) error {
	m.Lock()
	defer m.Unlock()

	clientId, err := client.GetId()
	if err != nil {
		return err
	}

	m.clients[*clientId] = client
	return nil
}
