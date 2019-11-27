package config

import (
	"fmt"
	"github.com/magiconair/properties"
	"os"
	"path/filepath"
)

var Props *properties.Properties

func getAppPath() string {
	appPath, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}
	return appPath
}

func init() {
	appPath := getAppPath()
	fmt.Println("appPath: ", appPath)

	workPath, err := os.Getwd()
	fmt.Println("workPath: ", workPath)

	appConfigPath := filepath.Join(workPath, "account.properties")

	Props, err = properties.LoadFile(appConfigPath, properties.UTF8)
	if err != nil {
		panic(err)
	}

	value, ok := Props.Get("account")
	if !ok {
		return
	}
	pwd, ok := Props.Get("password")
	if !ok {
		return
	}

	fmt.Println(value, pwd)
}

func GetValue(key string) string {
	val, ok := Props.Get(key)
	if !ok {
		panic(fmt.Sprintf("key: %s 在配置文件中没有配置", key))
	}
	return val
}
