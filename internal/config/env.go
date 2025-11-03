package config

import (
	"flag"
	"fmt"
	"reflect"

	"github.com/caarlos0/env/v11"
)

type ServerConfigEnv struct {
	STORE_INTERVAL    uint64      `env:"STORE_INTERVAL,notEmpty"`
	FILE_STORAGE_PATH string      `env:"FILE_STORAGE_PATH,notEmpty"`
	RESTORE           bool        `env:"RESTORE,notEmpty"`
	ADDRESS           HostAddress `env:"ADDRESS,notEmpty"`
}

type ServerConfig struct {
	envs            ServerConfigEnv
	Address         HostAddress
	StoreInterval   uint64
	FileStoragePath string
	Restore         bool

	paramAddress         HostAddress
	paramStoreInterval   uint64
	paramFileStoragePath string
	paramRestore         bool
}

func NewServerConfig() *ServerConfig {

	return &ServerConfig{}

}

func (se *ServerConfig) Init() {
	// fmt.Printf("start init:\n\n")
	// fmt.Println("======BEFORE PARAMS PARSE-----")
	// fmt.Printf("paramStoreInterval = %v\n", se.paramStoreInterval)
	// fmt.Printf("paramFileStoragePath = %v\n", se.paramFileStoragePath)
	// fmt.Printf("paramRestore = %v\n", se.paramRestore)
	// fmt.Printf("StoreInterval = %v\n", se.StoreInterval)
	// fmt.Printf("FileStoragePath = %v\n", se.FileStoragePath)
	// fmt.Printf("Restore = %v\n", se.Restore)
	flag.Var(&se.paramAddress, "a", "Net address host:port")
	flag.Uint64Var(&se.paramStoreInterval, "i", 300, "storeInterval")
	flag.StringVar(&se.paramFileStoragePath, "f", "save_file", "fileStoragePath")
	flag.BoolVar(&se.paramRestore, "r", false, "paramRestore")

	// fmt.Println("======AFTER PARAMS PARSE-----")
	// fmt.Printf("paramStoreInterval = %v\n", se.paramStoreInterval)
	// fmt.Printf("paramFileStoragePath = %v\n", se.paramFileStoragePath)
	// fmt.Printf("paramRestore = %v\n", se.paramRestore)
	// fmt.Printf("StoreInterval = %v\n", se.StoreInterval)
	// fmt.Printf("FileStoragePath = %v\n", se.FileStoragePath)
	// fmt.Printf("Restore = %v\n", se.Restore)
	// fmt.Printf("finish init:\n\n")
}

func (se *ServerConfig) Parse() {

	// fmt.Println("======BEFORE ENV PARSE-----")
	// fmt.Printf("paramStoreInterval = %v\n", se.paramStoreInterval)
	// fmt.Printf("paramFileStoragePath = %v\n", se.paramFileStoragePath)
	// fmt.Printf("paramRestore = %v\n", se.paramRestore)
	// fmt.Printf("StoreInterval = %v\n", se.StoreInterval)
	// fmt.Printf("FileStoragePath = %v\n", se.FileStoragePath)
	// fmt.Printf("Restore = %v\n", se.Restore)

	err := env.ParseWithOptions(&se.envs, env.Options{
		FuncMap: map[reflect.Type]env.ParserFunc{
			reflect.TypeOf(HostAddress{}): func(v string) (interface{}, error) {

				ha := NewHostAddress()
				err := ha.Set(v)

				return *ha, err
			},
		},
	})

	problemVars := make(map[string]bool)

	if err != nil {
		// fmt.Printf("err type %T:\n\n", err)
		if err, ok := err.(env.AggregateError); ok {
			// fmt.Printf("err.Errors: %v\n\n", err.Errors)

			for _, v := range err.Errors {
				fmt.Printf("err.Error: %T\n", v)
				fmt.Printf("err.Error: %v\n", v)

				if err1, ok := v.(env.EmptyVarError); ok {
					// fmt.Printf("err.EmptyVarError: %v\n", err1)
					// fmt.Printf("err.EmptyVarError.Key: %v\n", err1.Key)

					problemVars[err1.Key] = true
				}

				if err2, ok := v.(env.ParseError); ok {
					// fmt.Printf("err.ParseError: %v\n", err2)
					// fmt.Printf("err.ParseError.Name: %v\n", err2.Name)
					// fmt.Printf("err.ParseError.Type: %v\n", err2.Type)
					// fmt.Printf("err.ParseError.Err: %v\n", err2.Err)

					problemVars[err2.Name] = true
				}

				if _, ok := v.(HostAddressParseError); ok {

					problemVars["ADDRESS"] = true
				}

				//fmt.Println("----------------------")
			}

		}
	}

	fmt.Printf("problemVars = %v", problemVars)
	flag.Parse()

	// fmt.Println("======FLAG PARSED-----")
	// fmt.Printf("paramStoreInterval = %v\n", se.paramStoreInterval)
	// fmt.Printf("paramFileStoragePath = %v\n", se.paramFileStoragePath)
	// fmt.Printf("paramRestore = %v\n", se.paramRestore)
	// fmt.Printf("StoreInterval = %v\n", se.StoreInterval)
	// fmt.Printf("FileStoragePath = %v\n", se.FileStoragePath)
	// fmt.Printf("Restore = %v\n", se.Restore)

	if _, ok := problemVars["STORE_INTERVAL"]; !ok {
		se.StoreInterval = se.envs.STORE_INTERVAL
	} else {
		se.StoreInterval = se.paramStoreInterval
	}

	if _, ok := problemVars["FILE_STORAGE_PATH"]; !ok {
		se.FileStoragePath = se.envs.FILE_STORAGE_PATH
	} else {
		se.FileStoragePath = se.paramFileStoragePath
	}

	if _, ok := problemVars["RESTORE"]; !ok {
		se.Restore = se.envs.RESTORE
	} else {
		se.Restore = se.paramRestore
	}

	if _, ok := problemVars["ADDRESS"]; !ok {
		se.Address = se.envs.ADDRESS
	} else {
		se.Address = se.paramAddress
	}
	// fmt.Println("======RESULT-----")
	// fmt.Printf("paramStoreInterval = %v\n", se.paramStoreInterval)
	// fmt.Printf("paramFileStoragePath = %v\n", se.paramFileStoragePath)
	// fmt.Printf("paramRestore = %v\n", se.paramRestore)
	// fmt.Printf("StoreInterval = %v\n", se.StoreInterval)
	// fmt.Printf("FileStoragePath = %v\n", se.FileStoragePath)
	// fmt.Printf("Restore = %v\n", se.Restore)
}
