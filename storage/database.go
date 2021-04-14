///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Handles low level database control and interfaces

package storage

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/xx_network/primitives/id"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"sync"
	"time"
)

// DbTimeout determines maximum runtime (in seconds) of specific DB queries
const DbTimeout = 1

// Interface declaration for storage methods
type database interface {
	GetClient(id *id.ID) (*Client, error)
	UpsertClient(client *Client) error
}

// DatabaseImpl Struct implementing the database Interface with an underlying DB
type DatabaseImpl struct {
	db *gorm.DB // Stored database connection
}

// MapImpl Struct implementing the database Interface with an underlying Map
type MapImpl struct {
	clients map[id.ID]*Client
	sync.Mutex
}

// Client represents a User in Storage
type Client struct {
	Id []byte `gorm:"primaryKey"`

	// Base Key for message encryption
	BaseKey []byte `gorm:"not null"`

	// Used for Client registration
	PublicKey      []byte    `gorm:"not null"`
	Nonce          []byte    `gorm:"not null"`
	NonceTimestamp time.Time `gorm:"not null"`
	IsRegistered   bool      `gorm:"not null"`
}

// Initialize the database interface with database backend
// Returns a database interface, close function, and error
func newDatabase(username, password, dbName, address, port string, devMode bool) (database, error) {
	var err error
	var db *gorm.DB

	// Connect to the database if the correct information is provided
	if address != "" && port != "" {
		// Create the database connection
		connectString := fmt.Sprintf(
			"host=%s port=%s user=%s dbname=%s sslmode=disable",
			address, port, username, dbName)
		// Handle empty database password
		if len(password) > 0 {
			connectString += fmt.Sprintf(" password=%s", password)
		}
		db, err = gorm.Open(postgres.Open(connectString), &gorm.Config{
			Logger: logger.New(jww.TRACE, logger.Config{LogLevel: logger.Info}),
		})
	}

	// Return the map-backend interface
	// in the event there is a database error or information is not provided
	if (address == "" || port == "") || err != nil {

		var failReason string
		if err != nil {
			failReason = fmt.Sprintf("Unable to initialize database backend: %+v", err)
			jww.WARN.Printf(failReason)
		} else {
			failReason = "Database backend connection information not provided"
			jww.WARN.Printf(failReason)
		}

		if !devMode {
			jww.FATAL.Panicf("Cannot run in production "+
				"without a database: %s", failReason)
		}

		defer jww.INFO.Println("Map backend initialized successfully!")
		mapImpl := &MapImpl{
			clients: make(map[id.ID]*Client),
		}

		return database(mapImpl), nil
	}

	// Get and configure the internal database ConnPool
	sqlDb, err := db.DB()
	if err != nil {
		return database(&DatabaseImpl{}), errors.Errorf("Unable to configure database connection pool: %+v", err)
	}
	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	sqlDb.SetMaxIdleConns(10)
	// SetMaxOpenConns sets the maximum number of open connections to the Database.
	sqlDb.SetMaxOpenConns(100)
	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	sqlDb.SetConnMaxLifetime(24 * time.Hour)

	// Initialize the database schema
	// WARNING: Order is important. Do not change without database testing
	models := []interface{}{&Client{}}
	for _, model := range models {
		err = db.AutoMigrate(model)
		if err != nil {
			return database(&DatabaseImpl{}), err
		}
	}

	// Build the interface
	di := &DatabaseImpl{
		db: db,
	}

	jww.INFO.Println("Database backend initialized successfully!")
	return database(di), nil
}
