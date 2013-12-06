package dynamodb_test

import (
	simplejson "github.com/bitly/go-simplejson"
	"github.com/crowdmob/goamz/aws"
	"github.com/crowdmob/goamz/dynamodb"
	"testing"
)

func TestEmptyQuery(t *testing.T) {
	q := dynamodb.NewEmptyQuery()
	queryString := q.String()
	expectedString := "{}"

	if expectedString != queryString {
		t.Fatalf("Unexpected Query String : %s\n", queryString)
	}

}

func TestAddWriteRequestItems(t *testing.T) {
	auth := &aws.Auth{AccessKey: "", SecretKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"}
	server := dynamodb.Server{*auth, aws.USEast}
	primary := dynamodb.NewStringAttribute("WidgetFoo", "")
	secondary := dynamodb.NewNumericAttribute("Created", "")
	key := dynamodb.PrimaryKey{primary, secondary}
	table := server.NewTable("FooData", key)

	primary2 := dynamodb.NewStringAttribute("TestHashKey", "")
	secondary2 := dynamodb.NewNumericAttribute("TestRangeKey", "")
	key2 := dynamodb.PrimaryKey{primary2, secondary2}
	table2 := server.NewTable("TestTable", key2)

	q := dynamodb.NewEmptyQuery()

	attribute1 := dynamodb.NewNumericAttribute("testing", "4")
	attribute2 := dynamodb.NewNumericAttribute("testingbatch", "2111")
	attribute3 := dynamodb.NewStringAttribute("testingstrbatch", "mystr")
	item1 := []dynamodb.Attribute{*attribute1, *attribute2, *attribute3}

	attribute4 := dynamodb.NewNumericAttribute("testing", "444")
	attribute5 := dynamodb.NewNumericAttribute("testingbatch", "93748249272")
	attribute6 := dynamodb.NewStringAttribute("testingstrbatch", "myotherstr")
	item2 := []dynamodb.Attribute{*attribute4, *attribute5, *attribute6}

	attributeDel1 := dynamodb.NewStringAttribute("TestHashKeyDel", "DelKey")
	attributeDel2 := dynamodb.NewNumericAttribute("TestRangeKeyDel", "7777777")
	itemDel := []dynamodb.Attribute{*attributeDel1, *attributeDel2}

	attributeTest1 := dynamodb.NewStringAttribute("TestHashKey", "MyKey")
	attributeTest2 := dynamodb.NewNumericAttribute("TestRangeKey", "0193820384293")
	itemTest := []dynamodb.Attribute{*attributeTest1, *attributeTest2}

	tableItems := map[*dynamodb.Table]map[string][][]dynamodb.Attribute{}
	actionItems := make(map[string][][]dynamodb.Attribute)
	actionItems["Put"] = [][]dynamodb.Attribute{item1, item2}
	actionItems["Delete"] = [][]dynamodb.Attribute{itemDel}
	tableItems[table] = actionItems

	actionItems2 := make(map[string][][]dynamodb.Attribute)
	actionItems2["Put"] = [][]dynamodb.Attribute{itemTest}
	tableItems[table2] = actionItems2

	q.AddWriteRequestItems(tableItems)

	desiredString := "{\"RequestItems\":{\"FooData\":[{\"PutRequest\":{\"Item\":{\"testing\":{\"N\":\"4\"},\"testingbatch\":{\"N\":\"2111\"},\"testingstrbatch\":{\"S\":\"mystr\"}}}},{\"PutRequest\":{\"Item\":{\"testing\":{\"N\":\"444\"},\"testingbatch\":{\"N\":\"93748249272\"},\"testingstrbatch\":{\"S\":\"myotherstr\"}}}},{\"DeleteRequest\":{\"Key\":{\"TestHashKeyDel\":{\"S\":\"DelKey\"},\"TestRangeKeyDel\":{\"N\":\"7777777\"}}}}],\"TestTable\":[{\"PutRequest\":{\"Item\":{\"TestHashKey\":{\"S\":\"MyKey\"},\"TestRangeKey\":{\"N\":\"0193820384293\"}}}}]}}"
	queryString := q.String()

	if queryString != desiredString {
		t.Fatalf("Unexpected Query String : %s\n", queryString)
	}
}

func TestGetItemQuery(t *testing.T) {
	auth := &aws.Auth{AccessKey: "", SecretKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"}
	server := dynamodb.Server{*auth, aws.USEast}
	primary := dynamodb.NewStringAttribute("domain", "")
	key := dynamodb.PrimaryKey{primary, nil}
	table := server.NewTable("sites", key)

	q := dynamodb.NewQuery(table)
	q.AddKey(table, &dynamodb.Key{HashKey: "test"})

	queryString := []byte(q.String())

	json, err := simplejson.NewJson(queryString)

	if err != nil {
		t.Logf("JSON err : %s\n", err)
		t.Fatalf("Invalid JSON : %s\n", queryString)
	}

	tableName := json.Get("TableName").MustString()

	if tableName != "sites" {
		t.Fatalf("Expected tableName to be sites was : %s", tableName)
	}

	keyMap, err := json.Get("Key").Map()

	if err != nil {
		t.Fatalf("Expected a Key")
	}

	hashRangeKey := keyMap["domain"]

	if hashRangeKey == nil {
		t.Fatalf("Expected a HashKeyElement found : %s", keyMap)
	}

	if v, ok := hashRangeKey.(map[string]interface{}); ok {
		if val, ok := v["S"].(string); ok {
			if val != "test" {
				t.Fatalf("Expected HashKeyElement to have the value 'test' found : %s", val)
			}
		}
	} else {
		t.Fatalf("HashRangeKeyt had the wrong type found : %s", hashRangeKey)
	}
}

func TestUpdateQuery(t *testing.T) {
	auth := &aws.Auth{AccessKey: "", SecretKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"}
	server := dynamodb.Server{*auth, aws.USEast}
	primary := dynamodb.NewStringAttribute("domain", "")
	rangek := dynamodb.NewNumericAttribute("time", "")
	key := dynamodb.PrimaryKey{primary, rangek}
	table := server.NewTable("sites", key)

	countAttribute := dynamodb.NewNumericAttribute("count", "4")
	attributes := []dynamodb.Attribute{*countAttribute}

	q := dynamodb.NewQuery(table)
	q.AddKey(table, &dynamodb.Key{HashKey: "test", RangeKey: "1234"})
	q.AddUpdates(attributes, "ADD")

	queryString := []byte(q.String())

	json, err := simplejson.NewJson(queryString)

	if err != nil {
		t.Logf("JSON err : %s\n", err)
		t.Fatalf("Invalid JSON : %s\n", queryString)
	}

	tableName := json.Get("TableName").MustString()

	if tableName != "sites" {
		t.Fatalf("Expected tableName to be sites was : %s", tableName)
	}

	keyMap, err := json.Get("Key").Map()

	if err != nil {
		t.Fatalf("Expected a Key")
	}

	hashRangeKey := keyMap["domain"]

	if hashRangeKey == nil {
		t.Fatalf("Expected a HashKeyElement found : %s", keyMap)
	}

	rangeKey := keyMap["time"]

	if rangeKey == nil {
		t.Fatalf("Expected a RangeKeyElement found : %s", keyMap)
	}

}

func TestAddUpdates(t *testing.T) {
	auth := &aws.Auth{AccessKey: "", SecretKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"}
	server := dynamodb.Server{*auth, aws.USEast}
	primary := dynamodb.NewStringAttribute("domain", "")
	key := dynamodb.PrimaryKey{primary, nil}
	table := server.NewTable("sites", key)

	q := dynamodb.NewQuery(table)
	q.AddKey(table, &dynamodb.Key{HashKey: "test"})

	attr := dynamodb.NewStringSetAttribute("StringSet", []string{"str", "str2"})

	q.AddUpdates([]dynamodb.Attribute{*attr}, "ADD")
	queryString := []byte(q.String())

	json, err := simplejson.NewJson(queryString)

	if err != nil {
		t.Logf("JSON err : %s\n", err)
		t.Fatalf("Invalid JSON : %s\n", queryString)
	}

	attributeUpdates := json.Get("AttributeUpdates")
	if _, err := attributeUpdates.Map(); err != nil {
		t.Fatalf("Expected a AttributeUpdates found")
	}

	attributesModified := attributeUpdates.Get("StringSet")
	if _, err := attributesModified.Map(); err != nil {
		t.Fatalf("Expected a StringSet found : %s", err)
	}

	action := attributesModified.Get("Action")
	if v, err := action.String(); err != nil {
		t.Fatalf("Expected a action to be string : %s", err)
	} else if v != "ADD" {
		t.Fatalf("Expected a action to be ADD : %s", v)
	}

	value := attributesModified.Get("Value")
	if _, err := value.Map(); err != nil {
		t.Fatalf("Expected a Value found : %s", err)
	}

	string_set := value.Get("SS")
	string_set_ary, err := string_set.StringArray()
	if err != nil {
		t.Fatalf("Expected a string set found : %s", err)
	}
	if len(string_set_ary) != 2 {
		t.Fatalf("Expected a string set length to be 2 was : %d", len(string_set_ary))
	}

	for _, v := range string_set_ary {
		if v != "str" && v != "str2" {
			t.Fatalf("Expected a string to be str OR str2 was : %s", v)
		}
	}
}
