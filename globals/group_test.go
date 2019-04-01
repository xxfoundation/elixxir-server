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

func TestInitGroup_JSON(t *testing.T) {
	grp := InitGroup()

	json, err := grp.MarshalJSON()

	if err != nil {
		t.Errorf("Unable to marshall group to JSON")
	}

	jsonStr := string(json)

	expected := `{"gen":"5c7ff6b06f8f143fe8288433493e4769c4d988ace5be25a0e24809670716c613d7b0cee6932f8faa7c44d2cb24523da53fbe4f6ec3595892d1aa58c4328a06c46a15662e7eaa703a1decf8bbb2d05dbe2eb956c142a338661d10461c0d135472085057f3494309ffa73c611f78b32adbb5740c361c9f35be90997db2014e2ef5aa61782f52abeb8bd6432c4dd097bc5423b285dafb60dc364e8161f4a2a35aca3a10b1c4d203cc76a470a33afdcbdd92959859abd8b56e1725252d78eac66e71ba9ae3f1dd2487199874393cd4d832186800654760e1e34c09e4d155179f9ec0dc4473f996bdce6eed1cabed8b6f116f7ad9cf505df0f998e34ab27514b0ffe7","prime":"9db6fb5951b66bb6fe1e140f1d2ce5502374161fd6538df1648218642f0b5c48c8f7a41aadfa187324b87674fa1822b00f1ecf8136943d7c55757264e5a1a44ffe012e9936e00c1d3e9310b01c7d179805d3058b2a9f4bb6f9716bfe6117c6b5b3cc4d9be341104ad4a80ad6c94e005f4b993e14f091eb51743bf33050c38de235567e1b34c3d6a5c0ceaa1a0f368213c3d19843d0b4b09dcb9fc72d39c8de41f1bf14d4bb4563ca28371621cad3324b6a2d392145bebfac748805236f5ca2fe92b871cd8f9c36d3292b5509ca8caa77a2adfc7bfd77dda6f71125a7456fea153e433256a2261c6a06ed3693797e7995fad5aabbcfbe3eda2741e375404ae25b","primeQ":"f2c3119374ce76c9356990b465374a17f23f9ed35089bd969f61c6dde9998c1f"}`

	if expected != jsonStr {

		t.Errorf("Invalid group")
	}

}
