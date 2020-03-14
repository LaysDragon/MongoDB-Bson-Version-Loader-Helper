# BSON Migrator

For example please see [here](./example)

## Definition
The whole idea is base on a Migrator which will auto call the registered func to complete the transformer progress for need.

```go
package model

type User struct {
	Home                 string
	AddressButNotTHeSame string
	ANotherGender        string
	Age                  int
}

//for global usage latest version for this struct
var UserCurrentVersion = loader.NewVersionPanic("0.3")
//for global usage latest version struct for this struct
type UserCurrent = User_0_3

//v0.1 struct
type User_0_1 struct {
	Home    string
	Address string
	Gender  string
}

//v0.2 struct
type User_0_2 struct {
	Home     string
	XAddress string
	XGender  string
	Age      int
}

//v0.3 struct
type User_0_3 User

}
```
For auto migration from old version to new version struct with this migrator,you need to create a Registry with loader and transformer.

```go
package model
import loader "github.com/LaysDragon/bson-migrator"

//v0.1 loader
func User_0_3_Loader(src []byte, dst loader.VersionWrapper) error {
	dst.SetData(User_0_3{})//as custom loader,you need to set your data struct into wrapper for it to Unmarshal 

	if err := bson.Unmarshal(src, dst); err != nil {
		return err
	}
	return nil
}

//
func User_0_1_to_0_2_Transformer(container loader.VersionWrapper) error {
	if user_0_1, ok := container.GetData().(User_0_1); ok {
		user_0_2 := User_0_2{
			Home:     user_0_1.Home,
			XAddress: user_0_1.Address,
			XGender:  user_0_1.Address,
			Age:      0,
		}
		container.SetData(user_0_2)
		container.SetVersion(loader.NewVersionPanic("0.2"))
		return nil
	}
	return xerrors.Errorf("Cannot cast %T to %T:%w", container, User_0_1{}, loader.TransformerSrcTypeIncorrectError)

}

func User_0_2_to_0_3_Transformer(container loader.VersionWrapper) error {
	if user_0_2, ok := container.GetData().(User_0_2); ok {
		user_0_3 := User_0_3{
			Home:                 user_0_2.Home + "03version",
			AddressButNotTHeSame: user_0_2.XAddress + "new address format",
			ANotherGender:        user_0_2.XGender + "i am dragon",
			Age:                  user_0_2.Age + 25,
		}
		container.SetData(user_0_3)
		container.SetVersion(loader.NewVersionPanic("0.3"))
		return nil
	}
	return xerrors.Errorf("Cannot cast %T to %T:%w", container, User_0_2{}, loader.TransformerSrcTypeIncorrectError)

}

var UserLoadersRegistry = loader.NewRegistry(
	loader.SLoaders{//Loaders set version with responsible loader func
		"0.1": User_0_1{},//simply give a zero value struct,it will use default loader
   	    "0.2": loader.DefaultLoader(User_0_2{}) ,//same as above
	    "0.1": User_0_3_Loader,//use custom loader
	},
	loader.STransformers{//Transformer set with source version to target version with responsible transformer func
		"0.1": loader.STargetTransformers{
			"0.2": User_0_1_to_0_2_Transformer,
			...
		},
		"0.2": loader.STargetTransformers{
			"0.3": User_0_2_to_0_3_Transformer,
		},
	},
)
```
   
## Usage

```go
package model

func (s User) MarshalBSON() ([]byte, error) {
	//use versionCapture to wrapper before Marshal can add version inline field into bson data
	return bson.Marshal(loader.VersionCapture{Version: UserCurrentVersion, Data: s})
}

func (s *User) UnmarshalBSON(src []byte) error {
    //use registry Load method,it will load src byte into VersionWrapper with the specified struct you registered
	versionUser, err := UserLoadersRegistry.Load(src, UserCurrentVersion)
	if err != nil {
		return err
	}
    //extract it and cast back to current version
	*s = User(versionUser.GetData().(UserCurrent))
	return nil
}

```
```go
package main
import "go.mongodb.org/mongo-driver/bson"

func main(){
var bsonByte := ...//from mongodb go driver or something else
user:= User{}
bson.Unmarshal(bsonByte,&user)

}
```
