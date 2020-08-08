package igor

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const attributesFile = "_vals.json"

func MapToFiles(data map[string]interface{}, path string, perm os.FileMode) error {
	simple := map[string]interface{}{}

	for key, value := range data {
		switch value.(type) {
		case map[string]interface{}:
			nestedPath := path + "/" + key
			os.MkdirAll(nestedPath, perm)

			// can we do this without recusion?
			if err := MapToFiles(value.(map[string]interface{}), nestedPath, perm); err != nil {
				return err
			}
		default:
			simple[key] = value
		}
	}

	simpleData, err := json.Marshal(simple)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path+"/"+attributesFile, simpleData, perm)
}

func FilesToMap(path string) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	err := filepath.Walk(path, func(subPath string, info os.FileInfo, err error) error {
		subPath = strings.TrimPrefix(subPath, path)
		subPath = strings.TrimPrefix(subPath, "/")
		if len(subPath) == 0 {
			return nil
		}

		parts := strings.Split(subPath, "/")

		dataPath := data
		for i, part := range parts {
			if (i != len(parts)-1 && part != attributesFile) || (i != len(parts)-1 && info.IsDir()) {
				if _, exists := dataPath[part]; !exists {
					dataPath[part] = make(map[string]interface{})
				}
				dataPath = dataPath[part].(map[string]interface{})
			} else if i == len(parts)-1 && part == attributesFile {
				fileData, err := ioutil.ReadFile(path + "/" + subPath)
				if err != nil {
					return err
				}

				return json.Unmarshal(fileData, &dataPath)
			}
		}

		return nil
	})

	return data, err
}
