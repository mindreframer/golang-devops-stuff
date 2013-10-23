package dynamodb_test

import (
	"flag"
	"fmt"
	"github.com/crowdmob/goamz/aws"
	"github.com/crowdmob/goamz/dynamodb"
	"testing"
)

var amazon = flag.Bool("amazon", false, "Enable tests against amazon server")

func TestListTables(t *testing.T) {
	if !*amazon {
		t.Log("Amazon tests not enabled")
		return
	}

	auth, err := aws.EnvAuth()

	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	server := dynamodb.Server{auth, aws.USEast}

	tables, err := server.ListTables()

	if err != nil {
		t.Error(err.Error())
	}

	if len(tables) == 0 {
		t.Log("Expected table to be returned")
		t.FailNow()
	}

	fmt.Printf("tables %s\n", tables)

}

func TestCreateTable(t *testing.T) {
	if !*amazon {
		t.Log("Amazon tests not enabled")
		return
	}

	auth, err := aws.EnvAuth()

	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	server := dynamodb.Server{auth, aws.USEast}

	attr1 := dynamodb.AttributeDefinitionT{"TestHashKey", "S"}
	attr2 := dynamodb.AttributeDefinitionT{"TestRangeKey", "N"}

	tableName := "MyTestTable"

	keySch1 := dynamodb.KeySchemaT{"TestHashKey", "HASH"}
	keySch2 := dynamodb.KeySchemaT{"TestRangeKey", "RANGE"}

	provTPut := dynamodb.ProvisionedThroughputT{ReadCapacityUnits: 1, WriteCapacityUnits: 1}

	tdesc := dynamodb.TableDescriptionT{
		AttributeDefinitions:  []dynamodb.AttributeDefinitionT{attr1, attr2},
		TableName:             tableName,
		KeySchema:             []dynamodb.KeySchemaT{keySch1, keySch2},
		ProvisionedThroughput: provTPut,
	}

	status, err := server.CreateTable(tdesc)

	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}

	fmt.Println(status)

}

func TestGetItem(t *testing.T) {
	if !*amazon {
		t.Log("Amazon tests not enabled")
		return
	}

	auth, err := aws.EnvAuth()

	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	server := dynamodb.Server{auth, aws.USEast}
	primary := dynamodb.NewStringAttribute("domain", "")
	key := dynamodb.PrimaryKey{primary, nil}
	table := server.NewTable("production_storyarc-accelerator-sites",
		key)

	item, err := table.GetItem(&dynamodb.Key{HashKey: "ac-news.speedup.storytellerhq.com"})

	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	fmt.Printf("Item : %s\n", item)

}

func TestGetItemRange(t *testing.T) {
	if !*amazon {
		return
	}

	if !*amazon {
		t.Log("Amazon tests not enabled")
		return
	}

	auth, err := aws.EnvAuth()

	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	server := dynamodb.Server{auth, aws.USEast}
	primary := dynamodb.NewStringAttribute("uuid_type", "")
	rangeK := dynamodb.NewNumericAttribute("time", "")
	key := dynamodb.PrimaryKey{primary, rangeK}
	table := server.NewTable("production_storyarc-accelerator-analytics",
		key)

	item, err := table.GetItem(&dynamodb.Key{HashKey: "aee5df14-6961-4baa-bad1-a1150576594f_MISSES", RangeKey: "1348187524"})

	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	fmt.Printf("Item : %s\n", item)

}
