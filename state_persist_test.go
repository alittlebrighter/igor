package igor

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/buger/jsonparser"
	"github.com/stretchr/testify/assert"
)

var (
	data = map[string]interface{}{
		"one": 1,
		"two": map[string]interface{}{
			"nestedOne": "one",
			"nestedTwo": true,
		},
	}

	jsonData = []byte(`{
		"one": 1,
		"two": {
			"nestedOne": "one",
			"nestedTwo": true
		}
	}`)

	path = "testData"
)

func setup() error {
	return MapToFiles(data, path, 0755)
}

func cleanup() {
	os.RemoveAll(path)
}

func TestJsonToFiles(t *testing.T) {
	assert.Nil(t, JsonToFiles(jsonData, path, 0755))

	fileData, err := ioutil.ReadFile(path + "/two/" + attributesFile)
	assert.Nil(t, err)

	shouldBeStr, err := jsonparser.GetString(fileData, "nestedOne")
	assert.Equal(t, "one", shouldBeStr)

	cleanup()
}

func TestFilesToJson(t *testing.T) {
	setup()

	data, err := FilesToJson(path)
	assert.Nil(t, err)
	stuff, err := jsonparser.GetString(data, "two", "nestedOne")
	assert.Nil(t, err)
	assert.Equal(t, stuff, "one")

	cleanup()
}

func TestMapToFiles(t *testing.T) {
	assert.Nil(t, setup())
	assert.DirExists(t, path+"/two")

	cleanup()
}

func TestFilesToMap(t *testing.T) {
	assert.Nil(t, setup())
	ioutil.WriteFile(path+"/something.js", []byte(`{"one":2}`), 0755)

	readData, err := FilesToMap(path)
	_, shouldNotExist := readData["something"]

	assert.Nil(t, err)
	assert.EqualValues(t, 1, readData["one"])
	assert.Equal(t, false, shouldNotExist)
	assert.Equal(t, true, readData["two"].(map[string]interface{})["nestedTwo"])

	cleanup()
}
