package v1

import (
	loader "github.com/LaysDragon/MongodbVersionLoaderHelper"
	"go.mongodb.org/mongo-driver/bson"
)

type User struct {
	Home    string
	Address string
	Gender  string
}

var UserCurrentVersion = loader.NewVersionPanic("0.1")

type UserCurrent = User_0_1
type User_0_1 User

var UserLoadersRegistry = loader.NewRegistry(
	loader.SLoaders{
		"0.1": User_0_1{},
	},
	loader.STransformers{
		"0.1": loader.STargetTransformers{},
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
