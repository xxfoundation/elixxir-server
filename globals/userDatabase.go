package globals

import (
	"fmt"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	jww "github.com/spf13/jwalterweatherman"
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
	db *pg.DB
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

// Initialize the UserRegistry interface with a database backend
func InitDatabase() UserRegistry {
	db := pg.Connect(&pg.Options{
		User:     username,
		Password: password,
		Database: database,
		Addr:     address,
	})
	createSchema(db)
	return UserRegistry(&UserDatabase{
		db: db,
	})
}

// NewUser creates a new User object with default fields and given address.
func (m *UserDatabase) NewUser(address string) *User {
	// TODO
	return nil
}

// DeleteUser deletes a user with the given ID from userCollection.
func (m *UserDatabase) DeleteUser(id uint64) {
	// Perform the delete for the given ID
	user := UserDB{Id: id}
	err := m.db.Delete(&user)

	if err != nil {
		jww.ERROR.Printf("Unable to delete user %d! %v", id, err)
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
		return nil, false
	}
	// If we found a user for the given ID, return it
	return convertDbToUser(&user), true
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
func createSchema(db *pg.DB) {
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
			jww.FATAL.Println("Unable to create database schema!")
			panic(err)
		}
	}
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
func convertDbToUser(user *UserDB) (newUser *User) {
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
	return
}
