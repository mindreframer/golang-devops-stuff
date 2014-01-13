package dynamodb_test

import (
	"github.com/crowdmob/goamz/dynamodb"
	"launchpad.net/gocheck"
)

type ItemSuite struct {
	TableDescriptionT dynamodb.TableDescriptionT
	DynamoDBTest
	WithRange bool
}

func (s *ItemSuite) SetUpSuite(c *gocheck.C) {
	setUpAuth(c)
	s.DynamoDBTest.TableDescriptionT = s.TableDescriptionT
	s.server = &dynamodb.Server{dynamodb_auth, dynamodb_region}
	pk, err := s.TableDescriptionT.BuildPrimaryKey()
	if err != nil {
		c.Skip(err.Error())
	}
	s.table = s.server.NewTable(s.TableDescriptionT.TableName, pk)

	// Cleanup
	s.TearDownSuite(c)
	_, err = s.server.CreateTable(s.TableDescriptionT)
	if err != nil {
		c.Fatal(err)
	}
	s.WaitUntilStatus(c, "ACTIVE")
}

var item_suite = &ItemSuite{
	TableDescriptionT: dynamodb.TableDescriptionT{
		TableName: "DynamoDBTestMyTable",
		AttributeDefinitions: []dynamodb.AttributeDefinitionT{
			dynamodb.AttributeDefinitionT{"TestHashKey", "S"},
			dynamodb.AttributeDefinitionT{"TestRangeKey", "N"},
		},
		KeySchema: []dynamodb.KeySchemaT{
			dynamodb.KeySchemaT{"TestHashKey", "HASH"},
			dynamodb.KeySchemaT{"TestRangeKey", "RANGE"},
		},
		ProvisionedThroughput: dynamodb.ProvisionedThroughputT{
			ReadCapacityUnits:  1,
			WriteCapacityUnits: 1,
		},
	},
	WithRange: true,
}

var item_without_range_suite = &ItemSuite{
	TableDescriptionT: dynamodb.TableDescriptionT{
		TableName: "DynamoDBTestMyTable",
		AttributeDefinitions: []dynamodb.AttributeDefinitionT{
			dynamodb.AttributeDefinitionT{"TestHashKey", "S"},
		},
		KeySchema: []dynamodb.KeySchemaT{
			dynamodb.KeySchemaT{"TestHashKey", "HASH"},
		},
		ProvisionedThroughput: dynamodb.ProvisionedThroughputT{
			ReadCapacityUnits:  1,
			WriteCapacityUnits: 1,
		},
	},
	WithRange: false,
}

var _ = gocheck.Suite(item_suite)
var _ = gocheck.Suite(item_without_range_suite)

func (s *ItemSuite) TestPutGetDeleteItem(c *gocheck.C) {
	attrs := []dynamodb.Attribute{
		*dynamodb.NewStringAttribute("Attr1", "Attr1Val"),
	}

	var rk string
	if s.WithRange {
		rk = "1"
	}

	// Put
	if ok, err := s.table.PutItem("NewHashKeyVal", rk, attrs); !ok {
		c.Fatal(err)
	}

	// Get to verify Put operation
	pk := &dynamodb.Key{HashKey: "NewHashKeyVal", RangeKey: rk}
	item, err := s.table.GetItem(pk)
	if err != nil {
		c.Fatal(err)
	}

	if val, ok := item["TestHashKey"]; ok {
		c.Check(val, gocheck.DeepEquals, dynamodb.NewStringAttribute("TestHashKey", "NewHashKeyVal"))
	} else {
		c.Error("Expect TestHashKey to be found")
	}

	if s.WithRange {
		if val, ok := item["TestRangeKey"]; ok {
			c.Check(val, gocheck.DeepEquals, dynamodb.NewNumericAttribute("TestRangeKey", "1"))
		} else {
			c.Error("Expect TestRangeKey to be found")
		}
	}

	// Delete
	if ok, _ := s.table.DeleteItem(pk); !ok {
		c.Fatal(err)
	}

	// Get to verify Delete operation
	_, err = s.table.GetItem(pk)
	c.Check(err.Error(), gocheck.Matches, "Item not found")
}

func (s *ItemSuite) TestUpdateItem(c *gocheck.C) {
	attrs := []dynamodb.Attribute{
		*dynamodb.NewNumericAttribute("count", "0"),
	}

	var rk string
	if s.WithRange {
		rk = "1"
	}

	if ok, err := s.table.PutItem("NewHashKeyVal", rk, attrs); !ok {
		c.Fatal(err)
	}

	// UpdateItem with Add
	attrs = []dynamodb.Attribute{
		*dynamodb.NewNumericAttribute("count", "10"),
	}
	pk := &dynamodb.Key{HashKey: "NewHashKeyVal", RangeKey: rk}
	if ok, err := s.table.AddAttributes(pk, attrs); !ok {
		c.Error(err)
	}

	// Get to verify Add operation
	if item, err := s.table.GetItem(pk); err != nil {
		c.Error(err)
	} else {
		if val, ok := item["count"]; ok {
			c.Check(val, gocheck.DeepEquals, dynamodb.NewNumericAttribute("count", "10"))
		} else {
			c.Error("Expect count to be found")
		}
	}

	// UpdateItem with Put
	attrs = []dynamodb.Attribute{
		*dynamodb.NewNumericAttribute("count", "100"),
	}
	if ok, err := s.table.UpdateAttributes(pk, attrs); !ok {
		c.Error(err)
	}

	// Get to verify Put operation
	if item, err := s.table.GetItem(pk); err != nil {
		c.Fatal(err)
	} else {
		if val, ok := item["count"]; ok {
			c.Check(val, gocheck.DeepEquals, dynamodb.NewNumericAttribute("count", "100"))
		} else {
			c.Error("Expect count to be found")
		}
	}

	// UpdateItem with Delete
	attrs = []dynamodb.Attribute{
		*dynamodb.NewNumericAttribute("count", ""),
	}
	if ok, err := s.table.DeleteAttributes(pk, attrs); !ok {
		c.Error(err)
	}

	// Get to verify Delete operation
	if item, err := s.table.GetItem(pk); err != nil {
		c.Error(err)
	} else {
		if _, ok := item["count"]; ok {
			c.Error("Expect count not to be found")
		}
	}
}

func (s *ItemSuite) TestUpdateItemWithSet(c *gocheck.C) {
	attrs := []dynamodb.Attribute{
		*dynamodb.NewStringSetAttribute("list", []string{"A", "B"}),
	}

	var rk string
	if s.WithRange {
		rk = "1"
	}

	if ok, err := s.table.PutItem("NewHashKeyVal", rk, attrs); !ok {
		c.Error(err)
	}

	// UpdateItem with Add
	attrs = []dynamodb.Attribute{
		*dynamodb.NewStringSetAttribute("list", []string{"C"}),
	}
	pk := &dynamodb.Key{HashKey: "NewHashKeyVal", RangeKey: rk}
	if ok, err := s.table.AddAttributes(pk, attrs); !ok {
		c.Error(err)
	}

	// Get to verify Add operation
	if item, err := s.table.GetItem(pk); err != nil {
		c.Error(err)
	} else {
		if val, ok := item["list"]; ok {
			c.Check(val, gocheck.DeepEquals, dynamodb.NewStringSetAttribute("list", []string{"A", "B", "C"}))
		} else {
			c.Error("Expect count to be found")
		}
	}

	// UpdateItem with Delete
	attrs = []dynamodb.Attribute{
		*dynamodb.NewStringSetAttribute("list", []string{"A"}),
	}
	if ok, err := s.table.DeleteAttributes(pk, attrs); !ok {
		c.Error(err)
	}

	// Get to verify Delete operation
	if item, err := s.table.GetItem(pk); err != nil {
		c.Error(err)
	} else {
		if val, ok := item["list"]; ok {
			c.Check(val, gocheck.DeepEquals, dynamodb.NewStringSetAttribute("list", []string{"B", "C"}))
		} else {
			c.Error("Expect list to be remained")
		}
	}
}
