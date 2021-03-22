package util

import (
	"encoding/json"
	"io/ioutil"
)

// NewFileUtil creates new file util
func NewFileUtil(localDir string) *FileUtil {
	return &FileUtil{localDir}
}

// SaveBlockFile saves block file
func (f *FileUtil) SaveBlockFile(filename string, data interface{}) (bool, error) {
	filename = f.LocalDir + "/" + filename + ".json"
	saveData, err := json.Marshal(data)
	if err != nil {
		return false, err
	}

	err = ioutil.WriteFile(filename, saveData, 0644)
	if err != nil {
		return false, err
	}

	return true, nil
}

// GetJson read json file
func (f *FileUtil) GetJson(filename string) (json.RawMessage, error) {
	filename = f.LocalDir + "/" + filename + ".json"
	return ioutil.ReadFile(filename)
}

// Json2Struct transform json to struct
func Json2Struct(data json.RawMessage, resp interface{}) error {
	return json.Unmarshal(data, &resp)
}

// Struct2Json transform struct to json
func Struct2Json(theStruct interface{}) (json.RawMessage, error) {
	return json.Marshal(theStruct)
}
