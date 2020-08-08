package igor

import (
	"io/ioutil"
	"os"
	"testing"

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
	path = "testData"
)

func setup() error {
	return MapToFiles(data, path, 0755)
}

func cleanup() {
	os.RemoveAll(path)
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
