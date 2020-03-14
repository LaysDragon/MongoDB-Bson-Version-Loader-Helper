package v3

import (
	loader "github.com/LaysDragon/go-bson-migrator"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"
)

type User struct {
	Home                 string
	AddressButNotTHeSame string
	ANotherGender        string
	Age                  int
}

var UserCurrentVersion = loader.NewVersionPanic("0.3")

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

func User_0_2_Loader(src []byte, dst loader.VersionWrapper) error {
	dst.SetData(User_0_2{})

	if err := bson.Unmarshal(src, dst); err != nil {
		return err
	}
	return nil
}

type UserCurrent = User_0_3
type User_0_1 struct {
	Home    string
	Address string
	Gender  string
}

type User_0_2 struct {
	Home     string
	XAddress string
	XGender  string
	Age      int
}

type User_0_3 User

var UserLoadersRegistry = loader.NewRegistry(
	loader.SLoaders{
		"0.1": loader.DefaultLoader(User_0_1{}),
		"0.2": User_0_2_Loader,
		"0.3": User_0_3{},
	},
	loader.STransformers{
		"0.1": loader.STargetTransformers{
			"0.2": User_0_1_to_0_2_Transformer,
		},
		"0.2": loader.STargetTransformers{
			"0.3": User_0_2_to_0_3_Transformer,
		},
	},
)

func (s User) MarshalBSON() ([]byte, error) {

	return bson.Marshal(loader.VersionCapture{Version: UserCurrentVersion, Data: s})
}

func (s *User) UnmarshalBSON(src []byte) error {
	versionUser, err := UserLoadersRegistry.Load(src, UserCurrentVersion)
	if err != nil {
		return err
	}
	*s = User(versionUser.GetData().(UserCurrent))
	return nil
}
