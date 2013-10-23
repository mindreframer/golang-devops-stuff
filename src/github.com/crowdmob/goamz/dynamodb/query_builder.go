package dynamodb

import (
	"bytes"
	"fmt"
	"strconv"
)

type Query struct {
	buffer *bytes.Buffer
}

func NewEmptyQuery() *Query {
	q := &Query{new(bytes.Buffer)}
	q.buffer.WriteString("")
	return q
}

func NewQuery(t *Table) *Query {
	q := &Query{new(bytes.Buffer)}
	q.addTable(t)
	return q
}

// This way of specifing the key is used when doing a Get.
// If rangeKey is "", it is assumed to not want to be used
func (q *Query) AddKey(t *Table, key *Key) {
	b := q.buffer
	k := t.Key

	addComma(b)

	b.WriteString(quote("Key"))
	b.WriteString(":")

	b.WriteString("{")
	b.WriteString(quote(k.KeyAttribute.Name))
	b.WriteString(":")

	b.WriteString("{")
	b.WriteString(quote(k.KeyAttribute.Type))
	b.WriteString(":")
	b.WriteString(quote(key.HashKey))

	b.WriteString("}")

	if k.HasRange() {
		b.WriteString(",")
		b.WriteString(quote(k.RangeAttribute.Name))
		b.WriteString(":")

		b.WriteString("{")
		b.WriteString(quote(k.RangeAttribute.Type))
		b.WriteString(":")
		b.WriteString(quote(key.RangeKey))
		b.WriteString("}")
	}

	b.WriteString("}")
}

func (q *Query) addKeyAttributes(t *Table, key *Key) {
	b := q.buffer
	k := t.Key

	b.WriteString("{")
	b.WriteString(quote(k.KeyAttribute.Name))
	b.WriteString(":")

	b.WriteString("{")
	b.WriteString(quote(k.KeyAttribute.Type))
	b.WriteString(":")
	b.WriteString(quote(key.HashKey))

	b.WriteString("}")

	if k.HasRange() {
		b.WriteString(",")
		b.WriteString(quote(k.RangeAttribute.Name))
		b.WriteString(":")

		b.WriteString("{")
		b.WriteString(quote(k.RangeAttribute.Type))
		b.WriteString(":")
		b.WriteString(quote(key.RangeKey))
		b.WriteString("}")
	}

	b.WriteString("}")
}

func (q *Query) AddAttributesToGet(attributes []string) {
	if len(attributes) == 0 {
		return
	}
	b := q.buffer
	addComma(b)

	b.WriteString(quote("AttributesToGet"))
	b.WriteString(":")

	b.WriteString("[")

	for index, val := range attributes {
		if index > 0 {
			b.WriteString(",")
		}
		b.WriteString(quote(val))
	}

	b.WriteString("]")
}

func (q *Query) ConsistentRead(c bool) {
	if c == true {
		b := q.buffer
		addComma(b)

		b.WriteString(quote("ConsistentRead"))
		b.WriteString(":")
		b.WriteString("true")
	}
}

func (q *Query) AddGetRequestItems(tableKeys map[*Table][]Key) {
	b := q.buffer

	b.WriteString(quote("RequestItems"))
	b.WriteString(":")
	b.WriteString("{")

	firstItem := true
	for table, keys := range tableKeys {
		if !firstItem {
			b.WriteString(",")
		} else {
			firstItem = false
		}

		b.WriteString(quote(table.Name))
		b.WriteString(":")
		b.WriteString("{")

		b.WriteString(quote("Keys"))
		b.WriteString(":")
		b.WriteString("[")
		for index, key := range keys {
			if index > 0 {
				b.WriteString(",")
			}
			q.addKeyAttributes(table, &key)
		}
		b.WriteString("]")

		b.WriteString("}")
	}
	b.WriteString("}")
}

func (q *Query) AddWriteRequestItems(tableItems map[*Table]map[string][][]Attribute) {
	b := q.buffer

	b.WriteString(quote("RequestItems"))
	b.WriteString(":")
	b.WriteString("{")

	countTable := 0
	for table, itemActions := range tableItems {
		if countTable != 0 {
			b.WriteString(",")
		}
		countTable++

		b.WriteString(quote(table.Name))
		b.WriteString(":")
		b.WriteString("[")

		countAction := 0
		for action, items := range itemActions {
			if countAction != 0 {
				b.WriteString(",")
			}
			countAction++
			for index, attributes := range items {
				if index != 0 {
					b.WriteString(",")
				}

				b.WriteString("{")
				b.WriteString(quote(action+"Request"))
				b.WriteString(":")
				b.WriteString("{")

				if action == "Put" {
					b.WriteString(quote("Item"))
				} else {
					b.WriteString(quote("Key"))
				}
				b.WriteString(":")
				attributeList(b, attributes)

				b.WriteString("}")
				b.WriteString("}")
			}
		}

		b.WriteString("]")
	}
	b.WriteString("}")
}

func (q *Query) AddCreateRequestTable(description TableDescriptionT) {
	b := q.buffer

	b.WriteString(quote("AttributeDefinitions"))
	b.WriteString(":")
	b.WriteString("[")
	for i, attr := range description.AttributeDefinitions {
		if i != 0 {
			b.WriteString(",")
		}

		b.WriteString("{")
		b.WriteString(quote("AttributeName"))
		b.WriteString(":")
		b.WriteString(quote(attr.Name))
		b.WriteString(",")
		b.WriteString(quote("AttributeType"))
		b.WriteString(":")
		b.WriteString(quote(attr.Type))
		b.WriteString("}")
	}
	b.WriteString("]")
	b.WriteString(",")

	b.WriteString(quote("KeySchema"))
	b.WriteString(":")
	b.WriteString("[")
	for i, keyS := range description.KeySchema {
		if i != 0 {
			b.WriteString(",")
		}

		b.WriteString("{")
		b.WriteString(quote("AttributeName"))
		b.WriteString(":")
		b.WriteString(quote(keyS.AttributeName))
		b.WriteString(",")
		b.WriteString(quote("KeyType"))
		b.WriteString(":")
		b.WriteString(quote(keyS.KeyType))
		b.WriteString("}")
	}
	b.WriteString("]")
	b.WriteString(",")

	b.WriteString(quote("TableName"))
	b.WriteString(":")
	b.WriteString(quote(description.TableName))
	b.WriteString(",")

	b.WriteString(quote("ProvisionedThroughput"))
	b.WriteString(":")
	b.WriteString("{")
	b.WriteString(quote("ReadCapacityUnits"))
	b.WriteString(":")
	b.WriteString(strconv.Itoa(int(description.ProvisionedThroughput.ReadCapacityUnits)))
	b.WriteString(",")
	b.WriteString(quote("WriteCapacityUnits"))
	b.WriteString(":")
	b.WriteString(strconv.Itoa(int(description.ProvisionedThroughput.WriteCapacityUnits)))
	b.WriteString("}")

	// Todo: Implement LocalSecondayIndexes
}

func (q *Query) AddKeyConditions(comparisons []AttributeComparison) {
	b := q.buffer
	addComma(b)
	b.WriteString("\"KeyConditions\":{")
	q.addComparisons(comparisons)
	b.WriteString("}")
}

func (q *Query) AddLimit(limit int64) {
	b := q.buffer
	addComma(b)
	q.buffer.WriteString(keyValue("Limit", strconv.FormatInt(limit, 10)))
}
func (q *Query) AddSelect(value string) {
	b := q.buffer
	addComma(b)
	q.buffer.WriteString(keyValue("Select", value))
}

/*
   "ScanFilter":{
       "AttributeName1":{"AttributeValueList":[{"S":"AttributeValue"}],"ComparisonOperator":"EQ"}
   },
*/
func (q *Query) AddScanFilter(comparisons []AttributeComparison) {
	b := q.buffer
	addComma(b)
	b.WriteString("\"ScanFilter\":{")
	q.addComparisons(comparisons)
	b.WriteString("}")
}

func (q *Query) AddParallelScanConfiguration(segment int, totalSegments int) {
	b := q.buffer
	addComma(b)
	b.WriteString(fmt.Sprintf("\"Segment\":%d", segment))
	addComma(b)
	b.WriteString(fmt.Sprintf("\"TotalSegments\":%d", totalSegments))
}

func (q *Query) addComparisons(comparisons []AttributeComparison) {
	b := q.buffer
	for i, c := range comparisons {
		if i > 0 {
			b.WriteString(",")
		}

		b.WriteString(quote(c.AttributeName))
		b.WriteString(":{\"AttributeValueList\":[")
		for j, attributeValue := range c.AttributeValueList {
			if j > 0 {
				b.WriteString(",")
			}
			b.WriteString("{")
			b.WriteString(quote(attributeValue.Type))
			b.WriteString(":")
			b.WriteString(quote(attributeValue.Value))
			b.WriteString("}")
		}
		b.WriteString("], \"ComparisonOperator\":")
		b.WriteString(quote(c.ComparisonOperator))
		b.WriteString("}")
	}
}

// The primary key must be included in attributes.
func (q *Query) AddItem(attributes []Attribute) {
	b := q.buffer

	addComma(b)

	b.WriteString(quote("Item"))
	b.WriteString(":")

	attributeList(b, attributes)
}

func (q *Query) AddUpdates(attributes []Attribute, action string) {
	b := q.buffer

	addComma(b)

	b.WriteString(quote("AttributeUpdates"))
	b.WriteString(":")

	b.WriteString("{")
	for index, a := range attributes {
		if index > 0 {
			b.WriteString(",")
		}

		b.WriteString(quote(a.Name))
		b.WriteString(":")
		b.WriteString("{")
		b.WriteString(quote("Value"))
		b.WriteString(":")
		b.WriteString("{")
		b.WriteString(quote(a.Type))
		b.WriteString(":")

		if a.SetType() {
			b.WriteString("[")
			for i, aval := range a.SetValues {
				if i > 0 {
					b.WriteString(",")
				}
				b.WriteString(quote(aval))
			}
			b.WriteString("]")
		} else {
			b.WriteString(quote(a.Value))
		}

		b.WriteString("}")
		b.WriteString(",")
		b.WriteString(quote("Action"))
		b.WriteString(":")
		b.WriteString(quote(action))
		b.WriteString("}")
	}
	b.WriteString("}")
}

func (q *Query) AddExpected(attributes []Attribute) {
	b := q.buffer
	addComma(b)

	b.WriteString(quote("Expected"))
	b.WriteString(":")
	b.WriteString("{")

	for index, a := range attributes {
		if index > 0 {
			b.WriteString(",")
		}

		b.WriteString(quote(a.Name))
		b.WriteString(":")

		b.WriteString("{")

		if a.Exists != "" {
			b.WriteString(quote("Exists"))
			b.WriteString(":")
			b.WriteString("{")
			b.WriteString(quote(a.Exists))
			b.WriteString("}")
			b.WriteString(",")
		}

		b.WriteString(quote("Value"))
		b.WriteString(":")
		b.WriteString("{")
		b.WriteString(quote(a.Type))
		b.WriteString(":")

		if a.SetType() {
			b.WriteString("[")
			for i, aval := range a.SetValues {
				if i > 0 {
					b.WriteString(",")
				}
				b.WriteString(quote(aval))
			}
			b.WriteString("]")
		} else {
			b.WriteString(quote(a.Value))
		}

		b.WriteString("}")
		b.WriteString("}")
	}

	b.WriteString("}")
}

func attributeList(b *bytes.Buffer, attributes []Attribute) {
	b.WriteString("{")
	for index, a := range attributes {
		if index > 0 {
			b.WriteString(",")
		}

		b.WriteString(quote(a.Name))
		b.WriteString(":")

		b.WriteString("{")
		b.WriteString(quote(a.Type))
		b.WriteString(":")

		if a.SetType() {
			b.WriteString("[")
			for i, aval := range a.SetValues {
				if i > 0 {
					b.WriteString(",")
				}
				b.WriteString(quote(aval))
			}
			b.WriteString("]")
		} else {
			b.WriteString(strconv.Quote(a.Value)) // this needs to be quote escaped
		}

		b.WriteString("}")
	}
	b.WriteString("}")
}

func (q *Query) addTable(t *Table) {
	q.buffer.WriteString(keyValue("TableName", t.Name))
}

func quote(s string) string {
	return fmt.Sprintf("\"%s\"", s)
}

func addComma(b *bytes.Buffer) {
	if b.Len() != 0 {
		b.WriteString(",")
	}
}

func (q *Query) String() string {
	qs := fmt.Sprintf("{%s}", q.buffer.String())
	return qs
}
