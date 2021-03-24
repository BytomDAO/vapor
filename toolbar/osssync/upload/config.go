package upload

import (
	"encoding/json"
	"os"

	"github.com/bytom/bytom/errors"
)

// Config represent root of config
type Config struct {
	OssConfig *OssConfig `json:"oss_config"`
	VaporURL  string     `json:"vapor_url"`
}

// Oss logs cfg
type Login struct {
	Endpoint        string `json:"endpoint"`
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
}

// Oss cfg
type OssConfig struct {
	Login  *Login `json:"login"`
	Bucket string `json:"bucket"`
}

// LoadConfig read path file to the config object for Upload from Vapor to OSS
func LoadConfig(config interface{}) error {
	if len(os.Args) <= 1 {
		return errors.New("Please setup the config file path as Args[1]")
	}
	return LoadConfigByPath(os.Args[1], config)
}

// LoadConfigByPath read path file to the config object
func LoadConfigByPath(path string, config interface{}) error {
	configFile, err := os.Open(path)
	if err != nil {
		return errors.Wrap(err, "fail to open config file")
	}

	defer configFile.Close()
	return json.NewDecoder(configFile).Decode(config)
}
