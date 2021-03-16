package util

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

type FileUtil struct {
	localDir string
}

// NewFileUtil creates new file util
func NewFileUtil(localDir string) *FileUtil {
	return &FileUtil{localDir}
}

// SaveBlockFile saves block file
func (f *FileUtil) SaveBlockFile(filename string, data interface{}) (bool, error) {
	filename = f.localDir + "/" + filename + ".json"
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

// SaveTxtFile saves text file
func (f *FileUtil) SaveTxtFile(filename string, data string) (bool, error) {
	filename = f.localDir + "/" + filename + ".txt"
	saveData := []byte(data)
	err := ioutil.WriteFile(filename, saveData, 0644)
	if err != nil {
		return false, err
	}

	return true, nil
}

// GetJson read json file
func (f *FileUtil) GetJson(filename string) (json.RawMessage, error) {
	filename = f.localDir + "/" + filename + ".json"
	return ioutil.ReadFile(filename)
}

// RemoveLocal deletes file
func (f *FileUtil) RemoveLocal(filename string) error {
	return os.Remove(f.localDir + "/" + filename)
}

// Json2Struct transform json to struct
func Json2Struct(data json.RawMessage, resp interface{}) error {
	return json.Unmarshal(data, &resp)
}

// Struct2Json transform struct to json
func Struct2Json(theStruct interface{}) (json.RawMessage, error) {
	return json.Marshal(theStruct)
}

// IsExists if file or directory exist
func IsExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil && !os.IsExist(err) {
		return false
	}
	return true
}

// IfNoFileToCreate if the file is not exist, create the file
func IfNoFileToCreate(fileName string) (file *os.File) {
	var f *os.File
	var err error
	if !IsExists(fileName) {
		f, err = os.Create(fileName)
		if err != nil {
			return
		}

		defer f.Close()
	}
	return f
}
