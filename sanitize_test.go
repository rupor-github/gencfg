package gencfg

import (
	"fmt"
	"testing"
)

// NOTE: not save for multiple concurrent tests!
var count int

func (sanitizeTestFunctions) CountTestCall(name, data string) error {
	fmt.Println("CountTestCall with", name, data)
	count++
	return nil
}

type Config1 struct {
	Field1 string `yaml:"field1" sanitize:"test_call=CountTestCall"`
	Field2 string `yaml:"field2" sanitize:"test_call=CountTestCall"`
}

type Config2 struct {
	Dns []string `yaml:"dns" sanitize:"test_call=CountTestCall"`
}

type Config3 struct {
	Inside Config1 `yaml:"inside"`
}

type configAll struct {
	Field0          string              `yaml:"field0" sanitize:"test_call=CountTestCall"`
	Field00         *Config1            `yaml:"field00"`
	Uninitialized   *Config1            `yaml:"uninitialized"`
	SliceOfConfigs  []Config1           `yaml:"sliceOfConfigs"`
	SliceOfConfigs1 []*Config1          `yaml:"sliceOfConfigs1"`
	ArrayOfConfigs  [2]Config1          `yaml:"arrayOfConfigs"`
	ArrayOfConfigs1 [2]*Config1         `yaml:"arrayOfConfigs1"`
	MapOfConfigs    map[string]Config1  `yaml:"mapOfConfigs"`
	MapOfConfigs1   map[string]*Config1 `yaml:"mapOfConfigs1"`
	SliceOfStrings  []string            `yaml:"sliceOfStrings" sanitize:"test_call=CountTestCall"`
	SliceOfAny      []Config2           `yaml:"sliceOfAny"`
	StuctInStruct   Config3             `yaml:"stuctInStruct"`
	unsupported     *Config1            `yaml:"unsupported"`
}

var (
	c1 = Config1{"p11", "p12"}
	c2 = Config1{"p21", "p22"}
	c3 = Config1{"parr31", "parr32"}
	c4 = Config1{"parr41", "parr42"}
	c5 = Config1{"pmap51", "pmap52"}
	c6 = Config1{"pmap61", "pmap62"}
)

var testData = configAll{
	Field0: "data0",
	Field00: &Config1{
		Field1: "data01",
		Field2: "data02",
	},
	SliceOfConfigs: []Config1{
		{
			Field1: "data1",
			Field2: "data2",
		},
		{
			Field1: "data3",
			Field2: "data4",
		},
	},
	SliceOfConfigs1: []*Config1{&c1, &c2},
	ArrayOfConfigs: [2]Config1{
		{
			Field1: "data_arr1",
			Field2: "data_arr2",
		},
		{
			Field1: "data_arr3",
			Field2: "data_arr4",
		},
	},
	ArrayOfConfigs1: [2]*Config1{&c3, &c4},
	MapOfConfigs: map[string]Config1{
		"key1": {
			Field1: "data_map1",
			Field2: "data_map2",
		},
		"key2": {
			Field1: "data_map1",
			Field2: "data_map2",
		},
	},
	MapOfConfigs1:  map[string]*Config1{"key3": &c5, "key4": &c6},
	SliceOfStrings: []string{"s1", "s2"},
	SliceOfAny:     []Config2{{Dns: []string{"dns1", "dns2"}}, {Dns: []string{"dns3", "dns4"}}},
	StuctInStruct: Config3{
		Inside: Config1{"data_inside1", "data_inside2"},
	},
	unsupported: &c1,
}

func TestSanitize(t *testing.T) {
	t.Log("TestSanitize")
	const expected = 35
	if err := Sanitize(&testData); err != nil {
		t.Fatal(err)
	}
	if count != expected {
		t.Fatalf("count=%d, expected=%d", count, expected)
	}
	t.Logf("count=%d", count)
}
