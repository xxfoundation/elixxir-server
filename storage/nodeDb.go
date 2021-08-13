///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Handles the database ORM for nodes

package storage

import (
	"context"
	"errors"
	jww "github.com/spf13/jwalterweatherman"
	"git.xx.network/xx_network/primitives/id"
	"gorm.io/gorm"
	"time"
)

// Helper for forcing panics in the event of a CDE, otherwise acts as a pass-through
func catchCde(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		jww.FATAL.Panicf("Database call timed out: %+v", err.Error())
	}
	return err
}

// GetClient returns a Client from Database with the given ID
// Or an error if a matching Client does not exist
func (d *DatabaseImpl) GetClient(id *id.ID) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DbTimeout*time.Second)
	defer cancel()

	result := &Client{Id: id.Marshal()}
	err := d.db.WithContext(ctx).Take(result).Error
	return result, catchCde(err)
}

// UpsertClient inserts the given Client into Database if it does not exist
// Or updates the Database Client if its value does not match the given Client
func (d *DatabaseImpl) UpsertClient(client *Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), DbTimeout*time.Second)
	defer cancel()

	// Build a transaction to prevent race conditions
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Make a copy of the provided Client
		newClient := *client

		// Attempt to insert client into the Database,
		// or if it already exists, replace client with the Database value
		query := tx.FirstOrCreate(client, &Client{Id: client.Id})
		err := query.Error
		if err != nil {
			return err
		}

		// If client is already present in the Database, overwrite it with newClient
		if query.RowsAffected == 0 {
			return tx.Save(newClient).Error
		}

		// Commit
		return nil
	})
	return catchCde(err)
}
