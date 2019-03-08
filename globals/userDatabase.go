////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	"encoding/base64"
	"fmt"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/nonce"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/primitives/id"
	"sync"
	"time"
)

// Struct implementing the UserRegistry Interface with an underlying DB
type UserDatabase struct {
	db           *pg.DB                           // Stored database connection
	userChannels map[id.User]chan *pb.CmixMessage // Map of UserId to chan
}

// Struct representing a User in the database
type UserDB struct {
	// Overwrite table name
	tableName struct{} `sql:"users,alias:users"`

	// Convert between id.User and string using base64 StdEncoding
	Id string

	// Keys
	TransmissionBaseKey      []byte
	TransmissionRecursiveKey []byte
	ReceptionBaseKey         []byte
	ReceptionRecursiveKey    []byte

	// DSA Public Key
	PubKeyY []byte
	PubKeyP []byte
	PubKeyQ []byte
	PubKeyG []byte

	// Nonce
	Nonce          []byte
	NonceTimestamp time.Time
}

// Struct representing a Salt in the database
type SaltDB struct {
	// Overwrite table name
	tableName struct{} `sql:"salts,alias:salts"`

	// Primary key field containing the 256-bit salt
	Salt []byte
	// Contains the user id that the salt belongs to
	UserId string
}

func encodeUser(userId *id.User) string {
	return base64.StdEncoding.EncodeToString(userId.Bytes())
}

func decodeUser(userIdDB string) *id.User {
	userIdBytes, err := base64.StdEncoding.DecodeString(userIdDB)
	// This should only happen if you intentionally put invalid user ID
	// information in the database, which should never happen
	if err != nil {
		jww.ERROR.Print("decodeUser: Got error decoding user ID. " +
			"Returning zero ID instead")
		return id.ZeroID
	}

	return new(id.User).SetBytes(userIdBytes)
}

// Initialize the UserRegistry interface with appropriate backend
func NewUserRegistry(username, password,
	database, address string) UserRegistry {
	// Create the database connection
	db := pg.Connect(&pg.Options{
		User:        username,
		Password:    password,
		Database:    database,
		Addr:        address,
		PoolSize:    1,
		MaxRetries:  10,
		PoolTimeout: time.Duration(2) * time.Minute,
		IdleTimeout: time.Duration(10) * time.Minute,
		MaxConnAge:  time.Duration(1) * time.Hour,
	})
	// Attempt to connect to the database and initialize the schema
	err := createSchema(db)
	if err != nil {
		// Return the map-backed UserRegistry interface
		// in the event there is a database error
		jww.INFO.Println("Using map backend for UserRegistry!")

		uc := make(map[id.User]*User)
		salts := make(map[id.User][][]byte)

		return UserRegistry(&UserMap{
			userCollection: uc,
			saltCollection: salts,
			collectionLock: &sync.Mutex{},
		})
	} else {
		// Return the database-backed UserRegistry interface
		// in the event there are no database errors
		jww.INFO.Println("Using database backend for UserRegistry!")
		return UserRegistry(&UserDatabase{
			db:           db,
			userChannels: make(map[id.User]chan *pb.CmixMessage),
		})
	}
}

// Create dummy users to be manually inserted into the database
func PopulateDummyUsers() {
	// Deterministically create named users for demo
	for i := 0; i < NUM_DEMO_USERS; i++ {
		u := Users.NewUser()
		Users.UpsertUser(u)
	}
	// Named channel bot users
	for i := 0; i < NUM_DEMO_CHANNELS; i++ {
		u := Users.NewUser()
		Users.UpsertUser(u)
	}
}

// Inserts a unique salt into the salt table
// Returns true if successful, else false
func (m *UserDatabase) InsertSalt(userId *id.User, salt []byte) bool {
	// Convert id.User to string for database lookup
	userIdDB := encodeUser(userId)
	// Create a salt object with the given User
	s := SaltDB{UserId: userIdDB}

	// If the number of salts for the given UserId
	// is greater than the maximum allowed, then reject
	maxSalts := 300
	if count, _ := m.db.Model(&s).Count(); count > maxSalts {
		jww.ERROR.Printf("Unable to insert salt: Too many salts have already"+
			" been used for User %d", userId)
		return false
	}

	// Insert salt into the DB
	s.Salt = salt
	err := m.db.Insert(&s)

	// Verify there were no errors
	if err != nil {
		jww.ERROR.Printf("Unable to insert salt: %v", err)
		return false
	}

	return true
}

// NewUser creates a new User object with default fields and given address.
func (m *UserDatabase) NewUser() *User {
	newUser := UserRegistry(&UserMap{}).NewUser()
	return newUser
}

// DeleteUser deletes a user with the given ID from userCollection.
func (m *UserDatabase) DeleteUser(userId *id.User) {
	// Convert user ID to string for database lookup
	userIdDB := encodeUser(userId)
	// Perform the delete for the given ID
	user := UserDB{Id: userIdDB}
	err := m.db.Delete(&user)

	if err != nil {
		// Non-fatal error, user probably doesn't exist in the database
		jww.WARN.Printf("Unable to delete user %q! %v", userId, err)
	}
}

// GetUser returns a user with the given ID from user database
func (m *UserDatabase) GetUser(id *id.User) (user *User, err error) {
	// Perform the select for the given ID
	userIdDB := encodeUser(id)
	u := UserDB{Id: userIdDB}
	err = m.db.Select(&u)

	if err != nil {
		// If there was an error, no user for the given ID was found
		return nil, err
	}
	// If we found a user for the given ID, return it
	return m.convertDbToUser(&u), nil
}

// GetUser returns a user with a matching nonce from user database
func (m *UserDatabase) GetUserByNonce(nonce nonce.Nonce) (user *User, err error) {
	// Perform the select for the given nonce
	u := UserDB{Nonce: nonce.Bytes()}
	err = m.db.Select(&u)

	if err != nil {
		// If there was an error, no user for the given nonce was found
		return nil, err
	}
	// If we found a user for the given ID, return it
	return m.convertDbToUser(&u), nil
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
			"nick = EXCLUDED.nick," +
			"transmission_base_key = EXCLUDED.transmission_base_key," +
			"transmission_recursive_key = EXCLUDED.transmission_recursive_key," +
			"reception_base_key = EXCLUDED.reception_base_key," +
			"reception_recursive_key = EXCLUDED.reception_recursive_key," +
			"public_key = EXCLUDED.public_key").
		// Otherwise, insert the new user
		Insert()
	if err != nil {
		jww.ERROR.Printf("Unable to upsert user %q! %s", user.ID, err.Error())
	}
}

// CountUsers returns a count of the users in the database.
func (m *UserDatabase) CountUsers() int {
	count, err := m.db.Model(&UserDB{}).Count()
	if err != nil {
		jww.ERROR.Printf("Unable to count users! %s", err.Error())
		return 0
	}
	return count
}

// Create the database schema
func createSchema(db *pg.DB) error {
	for _, model := range []interface{}{&UserDB{}, &SaltDB{}} {
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
			jww.WARN.Printf("Unable to create database schema! %v", err)
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
		jww.WARN.Println(err)
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
	newUser.Id = encodeUser(user.ID)
	newUser.TransmissionBaseKey = user.Transmission.BaseKey.Bytes()
	newUser.TransmissionRecursiveKey = user.Transmission.RecursiveKey.Bytes()
	newUser.ReceptionBaseKey = user.Reception.BaseKey.Bytes()
	newUser.ReceptionRecursiveKey = user.Reception.RecursiveKey.Bytes()
	newUser.PubKeyY = user.PublicKey.GetY().Bytes()
	newUser.PubKeyP = user.PublicKey.GetP().Bytes()
	newUser.PubKeyQ = user.PublicKey.GetQ().Bytes()
	newUser.PubKeyG = user.PublicKey.GetG().Bytes()
	newUser.Nonce = user.Nonce.Bytes()
	newUser.NonceTimestamp = user.Nonce.GenTime
	return
}

// Convert UserDB type to User type
func (m *UserDatabase) convertDbToUser(user *UserDB) (newUser *User) {
	if user == nil {
		return nil
	}
	newUser = new(User)
	newUser.ID = decodeUser(user.Id)

	newUser.Transmission = ForwardKey{
		BaseKey:      cyclic.NewIntFromBytes(user.TransmissionBaseKey),
		RecursiveKey: cyclic.NewIntFromBytes(user.TransmissionRecursiveKey),
	}

	newUser.Reception = ForwardKey{
		BaseKey:      cyclic.NewIntFromBytes(user.ReceptionBaseKey),
		RecursiveKey: cyclic.NewIntFromBytes(user.ReceptionRecursiveKey),
	}

	newUser.PublicKey = signature.ReconstructPublicKey(
		signature.CustomDSAParams(cyclic.NewIntFromBytes(user.PubKeyP),
			cyclic.NewIntFromBytes(user.PubKeyQ),
			cyclic.NewIntFromBytes(user.PubKeyG)),
		cyclic.NewIntFromBytes(user.PubKeyY))

	newUser.Nonce = nonce.Nonce{
		GenTime:    user.NonceTimestamp,
		ExpiryTime: user.NonceTimestamp.Add(nonce.RegistrationTTL),
		TTL:        nonce.RegistrationTTL,
	}
	copy(user.Nonce, newUser.Nonce.Bytes())
	return
}
