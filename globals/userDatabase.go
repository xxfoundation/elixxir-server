////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"fmt"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/crypto/cyclic"
)

// Constants for the database connection
const (
	username = "jacobtaylor"
	password = ""
	database = "cmix_server"
	address  = ""
)

// Struct implementing the UserRegistry Interface with an underlying DB
type UserDatabase struct {
	db           *pg.DB                          // Stored database connection
	userChannels map[uint64]chan *pb.CmixMessage // Map of UserId to chan
}

// Struct representing a User in the database
type UserDB struct {
	// Overwrite table name
	tableName struct{} `sql:"users,alias:users"`

	Id      uint64
	Address string

	TransmissionBaseKey      []byte
	TransmissionRecursiveKey []byte
	ReceptionBaseKey         []byte
	ReceptionRecursiveKey    []byte

	PublicKey []byte
}

// Initialize the UserRegistry interface with appropriate backend
func newUserRegistry() UserRegistry {
	// Create the database connection
	db := pg.Connect(&pg.Options{
		User:     username,
		Password: password,
		Database: database,
		Addr:     address,
	})
	// Attempt to connect to the database and initialize the schema
	err := createSchema(db)

	if err != nil {
		// Return the map-backed UserRegistry interface
		// in the event there is a database error
		jww.INFO.Println("Using map backend for UserRegistry!")
		return UserRegistry(&UserMap{
			userCollection: make(map[uint64]*User),
		})
	} else {
		// Return the database-backed UserRegistry interface
		// in the event there are no database errors
		jww.INFO.Println("Using database backend for UserRegistry!")
		return UserRegistry(&UserDatabase{
			db:           db,
			userChannels: make(map[uint64]chan *pb.CmixMessage),
		})
	}
}

// NewUser creates a new User object with default fields and given address.
func (m *UserDatabase) NewUser(address string) *User {
	newUser := UserRegistry(&UserMap{}).NewUser(address)
	// Handle the conversion of the user's message buffer
	if userChannel, exists := m.userChannels[newUser.Id]; exists {
		// Add the old channel to the new User object if channel already exists
		newUser.MessageBuffer = userChannel
	} else {
		// Otherwise add the new channel to the userChannels map
		m.userChannels[newUser.Id] = newUser.MessageBuffer
	}
	return newUser
}

// DeleteUser deletes a user with the given ID from userCollection.
func (m *UserDatabase) DeleteUser(id uint64) {
	// Perform the delete for the given ID
	user := UserDB{Id: id}
	err := m.db.Delete(&user)

	if err != nil {
		// Non-fatal error, user probably doesn't exist in the database
		jww.DEBUG.Printf("Unable to delete user %d! %v", id, err)
	}
}

// GetUser returns a user with the given ID from userCollection
// and a boolean for whether the user exists
func (m *UserDatabase) GetUser(id uint64) (*User, bool) {
	// Perform the select for the given ID
	user := UserDB{Id: id}
	err := m.db.Select(&user)

	if err != nil {
		// If there was an error, no user for the given ID was found
		// So we will return nil, false, similar to map behavior
		jww.DEBUG.Printf("Unable to get user %d! %v", id, err)
		return nil, false
	}
	// If we found a user for the given ID, return it
	return m.convertDbToUser(&user), true
}

// UpsertUser inserts given user into the database or update the user if it
// already exists (Upsert operation).
func (m *UserDatabase) UpsertUser(user *User) {
	// Convert given user to database-friendly structure
	dbUser := convertUserToDb(user)
	// Perform the upsert
	_, err := m.db.Model(dbUser).
		// On conflict, update the user's fields
		OnConflict("(id) DO UPDATE").
		Set("address = EXCLUDED.address," +
			"transmission_base_key = EXCLUDED.transmission_base_key," +
			"transmission_recursive_key = EXCLUDED.transmission_recursive_key," +
			"reception_base_key = EXCLUDED.reception_base_key," +
			"reception_recursive_key = EXCLUDED.reception_recursive_key," +
			"public_key = EXCLUDED.public_key").
		// Otherwise, insert the new user
		Insert()
	if err != nil {
		jww.FATAL.Printf("Unable to upsert user %d!", user.Id)
		panic(err)
	}
}

// CountUsers returns a count of the users in the database.
func (m *UserDatabase) CountUsers() int {
	count, err := m.db.Model(&UserDB{}).Count()
	if err != nil {
		jww.FATAL.Println("Unable to count users!")
		panic(err)
	}
	return count
}

// Create the database schema
func createSchema(db *pg.DB) error {
	for _, model := range []interface{}{&UserDB{}} {
		err := db.CreateTable(model, &orm.CreateTableOptions{
			// Ignore create table if already exists?
			IfNotExists: true,
			// Create temporary table?
			Temp: false,
			// FKConstraints causes CreateTable to create foreign key constraints
			// for has one relations. ON DELETE hook can be added using tag
			// `sql:"on_delete:RESTRICT"` on foreign key field.
			FKConstraints: true,
			// Replaces PostgreSQL data type `text` with `varchar(n)`
			//Varchar: 255
		})
		if err != nil {
			// Return the error if one comes up
			jww.ERROR.Printf("Unable to create database schema! %v", err)
			return err
		}
	}
	// No error, return nil
	return nil
}

// Return given table information in string format
func getTableInfo(db *pg.DB, table string) string {
	// Return format for table information
	var info []struct {
		ColumnName string
		DataType   string
	}
	// Assemble the query
	query := fmt.Sprintf(`SELECT column_name,data_type FROM`+
		` information_schema.columns WHERE table_name = '%s';`, table)
	// Execute the query and insert into the struct
	_, err := db.Query(&info, query)
	// Verify there were no errors
	if err != nil {
		jww.ERROR.Println(err)
	}
	// Format the struct as a string and return
	return fmt.Sprintf("%s", info)
}

// Convert User type to UserDB type
func convertUserToDb(user *User) (newUser *UserDB) {
	if user == nil {
		return nil
	}
	newUser = new(UserDB)
	newUser.Id = user.Id
	newUser.Address = user.Address
	newUser.TransmissionBaseKey = user.Transmission.BaseKey.Bytes()
	newUser.TransmissionRecursiveKey = user.Transmission.RecursiveKey.Bytes()
	newUser.ReceptionBaseKey = user.Reception.BaseKey.Bytes()
	newUser.ReceptionRecursiveKey = user.Reception.RecursiveKey.Bytes()
	newUser.PublicKey = user.PublicKey.Bytes()
	return
}

// Convert UserDB type to User type
func (m *UserDatabase) convertDbToUser(user *UserDB) (newUser *User) {
	if user == nil {
		return nil
	}

	newUser = new(User)
	newUser.Id = user.Id
	newUser.Address = user.Address
	newUser.Transmission = ForwardKey{
		BaseKey:      cyclic.NewIntFromBytes(user.TransmissionBaseKey),
		RecursiveKey: cyclic.NewIntFromBytes(user.TransmissionRecursiveKey),
	}
	newUser.Reception = ForwardKey{
		BaseKey:      cyclic.NewIntFromBytes(user.ReceptionBaseKey),
		RecursiveKey: cyclic.NewIntFromBytes(user.ReceptionRecursiveKey),
	}
	newUser.PublicKey = cyclic.NewIntFromBytes(user.PublicKey)

	// Handle the conversion of the user's message buffer
	if userChannel, exists := m.userChannels[user.Id]; exists {
		// Add the channel to the new User object if it already exists
		newUser.MessageBuffer = userChannel
	} else {
		// Otherwise create a new channel for the new User object
		newUser.MessageBuffer = make(chan *pb.CmixMessage, 100)
		// And add it to the userChannels map
		m.userChannels[user.Id] = newUser.MessageBuffer
	}
	return
}
