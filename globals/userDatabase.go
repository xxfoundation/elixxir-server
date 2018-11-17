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
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/id"
	"sync"
	"time"
	"encoding/base64"
)

// Struct implementing the UserRegistry Interface with an underlying DB
type UserDatabase struct {
	db           *pg.DB                             // Stored database connection
	userChannels map[id.UserID]chan *pb.CmixMessage // Map of UserId to chan
}

// Struct representing a User in the database
type UserDB struct {
	// Overwrite table name
	tableName struct{} `sql:"users,alias:users"`

	// Convert between id.UserID and string using base64 StdEncoding
	Id      string
	Address string
	Nick    string

	TransmissionBaseKey      []byte
	TransmissionRecursiveKey []byte
	ReceptionBaseKey         []byte
	ReceptionRecursiveKey    []byte

	PublicKey []byte
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

func encodeUserID(userId *id.UserID) string {
	return base64.StdEncoding.EncodeToString(userId.Bytes())
}

func decodeUserID(userIdDB string) *id.UserID {
	userIdBytes, err := base64.StdEncoding.DecodeString(userIdDB)
	// This should only happen if you intentionally put invalid user ID
	// information in the database, which should never happen
	if err != nil {
		jww.ERROR.Print("decodeUserID: Got error decoding user ID. " +
			"Returning zero ID instead")
		return id.ZeroID
	}

	return new(id.UserID).SetBytes(userIdBytes)
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

		uc := make(map[id.UserID]*User)
		salts := make(map[id.UserID][][]byte)

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
			userChannels: make(map[id.UserID]chan *pb.CmixMessage),
		})
	}
}

// TODO: remove or improve this
// Create dummy users to be manually inserted into the database
func PopulateDummyUsers() {

	nickList := []string{"David", "Jim", "Ben", "Rick", "Spencer", "Jake",
		"Mario", "Will", "Sydney", "Jono"}
	channelList := []string{"#General", "#Engineering", "#Lunch", "#Random"}

	// Deterministically create named users for demo
	for i := 0; i < len(nickList); i++ {
		u := Users.NewUser("")
		u.Nick = nickList[i]
		Users.UpsertUser(u)
	}
	// Extra un-named users for demo expansion
	for i := len(nickList) + 1; i <= NUM_DEMO_USERS; i++ {
		u := Users.NewUser("")
		u.Nick = ""
		Users.UpsertUser(u)
	}
	// Named channel bot users
	for i := 0; i < len(channelList); i++ {
		u := Users.NewUser("")
		u.Nick = channelList[i]
		Users.UpsertUser(u)
	}
	// Extra un-named users for demo expansion
	for i := len(channelList) + 1; i <= NUM_DEMO_CHANNELS; i++ {
		u := Users.NewUser("")
		u.Nick = ""
		Users.UpsertUser(u)
	}
}

// Inserts a unique salt into the salt table
// Returns true if successful, else false
func (m *UserDatabase) InsertSalt(userId *id.UserID, salt []byte) bool {
	// Convert id.UserID to string for database lookup
	userIdDB := encodeUserID(userId)
	// Create a salt object with the given UserID
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
func (m *UserDatabase) NewUser(address string) *User {
	newUser := UserRegistry(&UserMap{}).NewUser(address)
	return newUser
}

// DeleteUser deletes a user with the given ID from userCollection.
func (m *UserDatabase) DeleteUser(userId *id.UserID) {
	// Convert user ID to string for database lookup
	userIdDB := encodeUserID(userId)
	// Perform the delete for the given ID
	user := UserDB{Id: userIdDB}
	err := m.db.Delete(&user)

	if err != nil {
		// Non-fatal error, user probably doesn't exist in the database
		jww.WARN.Printf("Unable to delete user %q! %v", userId, err)
	}
}

// GetUser returns a user with the given ID from userCollection
// and a boolean for whether the user exists
func (m *UserDatabase) GetUser(userId *id.UserID) (*User, error) {
	// Perform the select for the given ID
	userIdDB := encodeUserID(userId)
	user := UserDB{Id: userIdDB}
	err := m.db.Select(&user)

	if err != nil {
		// If there was an error, no user for the given ID was found
		// So we will return nil, false, similar to map behavior
		return nil, err
	}
	// If we found a user for the given ID, return it
	return m.convertDbToUser(&user), nil
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
	newUser.Id = encodeUserID(user.ID)
	newUser.Address = user.Address
	newUser.Nick = user.Nick
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
	newUser.ID = decodeUserID(user.Id)
	newUser.Address = user.Address
	newUser.Nick = user.Nick
	newUser.Transmission = ForwardKey{
		BaseKey:      cyclic.NewIntFromBytes(user.TransmissionBaseKey),
		RecursiveKey: cyclic.NewIntFromBytes(user.TransmissionRecursiveKey),
	}
	newUser.Reception = ForwardKey{
		BaseKey:      cyclic.NewIntFromBytes(user.ReceptionBaseKey),
		RecursiveKey: cyclic.NewIntFromBytes(user.ReceptionRecursiveKey),
	}
	newUser.PublicKey = cyclic.NewIntFromBytes(user.PublicKey)

	return
}
