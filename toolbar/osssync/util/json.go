package util

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
)

type FileUtil struct {
	localDir string
}

const LOCALDIR = "./blocks"

func NewFileUtil() *FileUtil {
	return &FileUtil{LOCALDIR}
}

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

func (f *FileUtil) SaveTxtFile(filename string, data string) (bool, error) {
	filename = f.localDir + "/" + filename + ".txt"
	saveData := []byte(data)
	err := ioutil.WriteFile(filename, saveData, 0644)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (f *FileUtil) GetJson(filename string) (json.RawMessage, error) {
	filename = f.localDir + "/" + filename + ".json"
	return ioutil.ReadFile(filename)
}

func (f *FileUtil) RemoveLocal(filename string) error {
	return os.Remove(f.localDir + "/" + filename)
}

func Json2Struct(data json.RawMessage, resp interface{}) error {
	return json.Unmarshal(data, &resp)
}

func Struct2Json(theStruct interface{}) (json.RawMessage, error) {
	return json.Marshal(theStruct)
}


// IsExists 判断所给路径文件/文件夹是否存在
func IsExists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil && !os.IsExist(err) {
		return false
	}
	return true
}

// IfNoFileToCreate 文件不存在就创建文件
func IfNoFileToCreate(fileName string) (file *os.File) {
	var f *os.File
	var err error
	if !IsExists(fileName) {
		f, err = os.Create(fileName)
		if err != nil {
			return
		}
		log.Printf("IfNoFileToCreate 函数成功创建文件:%s", fileName)
		defer f.Close()
	}
	return f
}

