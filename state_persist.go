package igor

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/buger/jsonparser"
)

const attributesFile = "_vals.json"

func JsonToFiles(data []byte, path string, perm os.FileMode) error {
	attributes := []byte("{}")
	basePath := ""
	if len(path) > 0 {
		basePath = path + "/"
	}

	err := jsonparser.ObjectEach(data, func(key []byte, value []byte, dataType jsonparser.ValueType, offset int) error {
		switch dataType {
		case jsonparser.Object:
			nestedPath := basePath + string(key)
			os.MkdirAll(nestedPath, perm)

			// can we do this without recursion?
			if err := JsonToFiles(value, nestedPath, perm); err != nil {
				return err
			}
		default:
			if dataType == jsonparser.String {
				value = append([]byte(`"`), value...)
				value = append(value, byte('"'))
			}
			var err error
			attributes, err = jsonparser.Set(attributes, value, string(key))
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	if len(attributes) > 2 { // empty attributes == "{}"
		return ioutil.WriteFile(basePath+attributesFile, attributes, perm)
	}
	return nil
}

func FilesToJson(path string) ([]byte, error) {
	data := []byte("{}")

	err := filepath.Walk(path, func(subPath string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		subPath = strings.TrimPrefix(subPath, path)
		if len(subPath) == 0 {
			return nil
		}
		subPath = strings.TrimLeft(subPath, "/")

		parts := strings.Split(subPath, "/")

		if info.IsDir() || (len(parts) > 0 && parts[len(parts)-1] != attributesFile) {
			return nil
		}

		fileData, err := ioutil.ReadFile(path + "/" + subPath)
		if err != nil {
			return err
		}

		if len(parts) == 1 && parts[0] == attributesFile {
			// if we have the attributes file at the root of the specified path,
			// unmarshal to verify we have valid JSON
			var attributes map[string]json.RawMessage
			err := json.Unmarshal(fileData, &attributes)
			if err != nil {
				return err
			}

			for k, v := range attributes {
				data, err = jsonparser.Set(data, v, k)
				if err != nil {
					return err
				}
			}

			return nil
		}

		data, err = jsonparser.Set(data, fileData, parts[0:len(parts)-1]...)
		return err
	})

	return data, err
}

func MapToFiles(data map[string]interface{}, path string, perm os.FileMode) error {
	simple := map[string]interface{}{}
	basePath := ""
	if len(path) > 0 {
		basePath = path + "/"
	}

	for key, value := range data {
		switch value.(type) {
		case map[string]interface{}:
			nestedPath := basePath + key
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

	return ioutil.WriteFile(basePath+attributesFile, simpleData, perm)
}

func FilesToMap(path string) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	err := filepath.Walk(path, func(subPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

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
