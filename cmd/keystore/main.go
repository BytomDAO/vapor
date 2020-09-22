package main

import (
	"fmt"
	"os"
	"encoding/json"
)

type PersonInfo struct {
	Name    string
	age     int32
	Sex     bool
	Hobbies []string
}

func readFile() {

	filePtr, err := os.Open("person_info.json")
	if err != nil {
		fmt.Println("Open file failed [Err:%s]", err.Error())
		return
	}
	defer filePtr.Close()

	var person []PersonInfo

	// 创建json解码器
	decoder := json.NewDecoder(filePtr)
	err = decoder.Decode(&person)
	if err != nil {
		fmt.Println("Decoder failed", err.Error())

	} else {
		fmt.Println("Decoder success")
		fmt.Println(person)
	}
}

func main() {
	fmt.Println("Hello world!")
	readFile()
}
