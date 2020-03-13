package v1

import (
	"encoding/json"
	"fmt"
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
	user := User{
		Home:    "home sweet home",
		Address: "address1",
		Gender:  "dog",
	}
	if userBson, err := bson.Marshal(user); err != nil {
		t.Logf("%+v", err)
		return
	} else {
		userMap := map[string]interface{}{}
		if err := bson.Unmarshal(userBson, userMap); err != nil {
			t.Logf("%+v", err)
			return
		}

		if err := PrintJson(userMap); err != nil {
			t.Logf("%+v", err)
			return
		}

	}
}
