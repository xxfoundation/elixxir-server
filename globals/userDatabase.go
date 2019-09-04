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
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/nonce"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/id"
	"time"
)

// Structure implementing the UserRegistry Interface with an underlying DB.
type UserDatabase struct {
	db *pg.DB
}

// Structure representing a User in the database.
type UserDB struct {
	// Overwrite table name
	tableName struct{} `sql:"users,alias:users"`

	// Convert between id.User and string using base64 StdEncoding
	Id string

	// Base Key for message encryption
	BaseKey []byte
	// RSA Public Key for Client Registration
	RsaPublicKey []byte

	// Nonce
	Nonce          []byte
	NonceTimestamp time.Time

	//Registration flag
	IsRegistered bool
}

// Structure representing a Salt in the database.
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
		err = errors.New(err.Error())
		jww.ERROR.Printf("decodeUser: Got error decoding user ID: %+v,"+
			" Returning zero ID instead", err)
		return id.ZeroID
	}

	return id.NewUserFromBytes(userIdBytes)
}

// Initialize the UserRegistry interface with appropriate backend
func NewUserRegistry(username, password,
	database, address string) UserRegistry {
	// Create the database connection
	db := pg.Connect(&pg.Options{
		User:         username,
		Password:     password,
		Database:     database,
		Addr:         address,
		MaxRetries:   10,
		MinIdleConns: 1,
	})

	// Attempt to connect to the database and initialize the schema
	err := createSchema(db)
	if err != nil {
		// Return the map-backed UserRegistry interface
		// in the event there is a database error
		jww.ERROR.Printf("Unable to initalize database backend: %+v", 
		    errors.New(err.Error())
		jww.INFO.Println("Using map backend for UserRegistry!")
		return UserRegistry(&UserMap{})
	} else {
		// Return the database-backed UserRegistry interface
		// in the event there are no database errors
		jww.INFO.Println("Using database backend for UserRegistry!")
		return UserRegistry(&UserDatabase{db: db})
	}
}

// Inserts a unique salt into the salt table
// Returns true if successful, else false
func (m *UserDatabase) InsertSalt(userId *id.User, salt []byte) error {
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
		return errors.New(fmt.Sprintf(errTooManySalts, userId))
	}

	// Insert salt into the DB
	s.Salt = salt
	err := m.db.Insert(&s)

	// Verify there were no errors
	if err != nil {
		err = errors.New(err.Error())
		jww.ERROR.Printf("Unable to insert salt: %v", err)
	}

	return err
}

// NewUser creates a new User object with default fields and given address.
func (m *UserDatabase) NewUser(grp *cyclic.Group) *User {
	newUser := UserRegistry(&UserMap{}).NewUser(grp)
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
		err = errors.New(err.Error())
		jww.WARN.Printf("Unable to delete user %q! %+v", userId, err)
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
		jww.ERROR.Printf("Unable to find user %v", id)
		return nil, errors.New(err.Error())
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
		Set("base_key = EXCLUDED.base_key," +
			"rsa_public_key = EXCLUDED.rsa_public_key," +
			"nonce = EXCLUDED.nonce," +
			"nonce_timestamp = EXCLUDED.nonce_timestamp," +
			"is_registered = EXCLUDED.is_registered").
		// Otherwise, insert the new user
		Insert()
	if err != nil {
		err = errors.New(err.Error())
		jww.ERROR.Printf("Unable to upsert user %q! %+v", user.ID, err)
	}
}

// CountUsers returns a count of the users in the database.
func (m *UserDatabase) CountUsers() int {
	count, err := m.db.Model(&UserDB{}).Count()
	if err != nil {
		err = errors.New(err.Error())
		jww.ERROR.Printf("Unable to count users! %+v", err)
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
			err = errors.New(err.Error())
			jww.WARN.Printf("Unable to create database schema! %+v", err)
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
	newUser.BaseKey = user.BaseKey.Bytes()

	pubKeyBytes := make([]byte, 0)

	if user.RsaPublicKey != nil {
		pubKeyBytes = rsa.CreatePublicKeyPem(user.RsaPublicKey)
	}

	newUser.RsaPublicKey = pubKeyBytes
	newUser.Nonce = user.Nonce.Bytes()
	newUser.NonceTimestamp = user.Nonce.GenTime
	newUser.IsRegistered = user.IsRegistered
	return
}

// Convert UserDB type to User type
func (m *UserDatabase) convertDbToUser(user *UserDB) (newUser *User) {
	if user == nil {
		return nil
	}
	newUser = new(User)
	newUser.ID = decodeUser(user.Id)
	newUser.BaseKey = grp.NewIntFromBytes(user.BaseKey)
	newUser.IsRegistered = user.IsRegistered

	if user.RsaPublicKey != nil && len(user.RsaPublicKey) != 0 {
		rsaPublicKey, err := rsa.LoadPublicKeyFromPem(user.RsaPublicKey)
		if err != nil {
			jww.ERROR.Printf("Unable to convert PEM to public key: %+v\n%+v",
				user.RsaPublicKey, errors.New(err.Error()))
		}
		newUser.RsaPublicKey = rsaPublicKey
	}

	newUser.Nonce = nonce.Nonce{
		GenTime:    user.NonceTimestamp,
		ExpiryTime: user.NonceTimestamp.Add(nonce.RegistrationTTL * time.Second),
		TTL:        nonce.RegistrationTTL * time.Second,
	}
	copy(newUser.Nonce.Value[:], user.Nonce)

	return
}
