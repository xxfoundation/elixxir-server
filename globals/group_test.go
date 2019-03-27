package globals

import (
	"fmt"
	"reflect"
	"testing"
)

// Tests that the group that is set is the same that is retrieved.
func TestSetGroup_GetGroup(t *testing.T) {
	InitCrypto()

	SetGroup(Group)

	if !reflect.DeepEqual(GetGroup(), Group) {
		t.Errorf("The group returned by GetGroup() does not match the set group\n\trecieved: %#v\n\texpected:%v", GetGroup(), Group)
	}
}

// Tests that SetGroup() panics when setting the group a second time.
func TestSetGroup_Again(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in SetGroup(): ", r)
		} else {
			t.Errorf("SetGroup() did not panick when expected while attempting to set the group again")
		}
	}()

	InitCrypto()

	SetGroup(Group)

	SetGroup(Group)
}
