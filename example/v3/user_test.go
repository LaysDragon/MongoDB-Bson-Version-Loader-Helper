package v3

import (
	"encoding/json"
	"fmt"
	loader "github.com/LaysDragon/MongodbVersionLoaderHelper"
	"go.mongodb.org/mongo-driver/bson"
	"testing"
)

func PrintJson(v interface{}) error {
	if result, err := json.Marshal(v); err != nil {
		return err
	} else {
		fmt.Println(string(result))
	}
	return nil
}

func TestUser(t *testing.T) {
	userOld := User_0_1{
		Home:    "home sweet home",
		Address: "address1",
		Gender:  "dog",
	}
	//模擬帶版本的bson {version:"0.1",home:"...",address:"...",...}，從mongodb讀出來的
	if userOldBson, err := bson.Marshal(loader.VersionCapture{Version: loader.NewVersionPanic("0.1"), Data: userOld}); err != nil {
		t.Logf("%+v", err)
		return
	} else {
		user := User{}

		if err := bson.Unmarshal(userOldBson, &user); err != nil {
			t.Logf("%+v", err)
			return
		}

		if err := PrintJson(user); err != nil {
			t.Logf("%+v", err)
			return
		}

	}
}
