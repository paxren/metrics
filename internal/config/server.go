package config

import (
	"flag"
	"fmt"
	"reflect"

	"github.com/caarlos0/env/v11"
)

type ServerConfigEnv struct {
	StoreInterval   uint64      `env:"STORE_INTERVAL,notEmpty"`
	FileStoragePath string      `env:"FILE_STORAGE_PATH,notEmpty"`
	DatabaseDSN     string      `env:"DATABASE_DSN,notEmpty"`
	Restore         bool        `env:"RESTORE,notEmpty"`
	Address         HostAddress `env:"ADDRESS,notEmpty"`
	Key             string      `env:"KEY,notEmpty"`
	AuditFile       string      `env:"AUDIT_FILE,notEmpty"`
	AuditURL        string      `env:"AUDIT_URL,notEmpty"`
}

type ServerConfig struct {
	envs            ServerConfigEnv
	Address         HostAddress
	StoreInterval   uint64
	FileStoragePath string
	DatabaseDSN     string
	Key             string
	Restore         bool
	AuditFile       string
	AuditURL        string

	paramAddress         HostAddress
	paramStoreInterval   uint64
	paramFileStoragePath string
	paramDatabaseDSN     string
	paramKey             string
	paramRestore         bool
	paramAuditFile       string
	paramAuditURL        string
}

func NewServerConfig() *ServerConfig {

	return &ServerConfig{
		paramAddress: *NewHostAddress(),
	}

}

func (se *ServerConfig) Init() {
	// fmt.Printf("start init:\n\n")
	// fmt.Println("======BEFORE PARAMS PARSE-----")
	// fmt.Printf("paramStoreInterval = %v\n", se.paramStoreInterval)
	// fmt.Printf("paramFileStoragePath = %v\n", se.paramFileStoragePath)
	// fmt.Printf("paramRestore = %v\n", se.paramRestore)
	// fmt.Printf("paramAdress = %v\n", se.paramAddress)
	// fmt.Printf("StoreInterval = %v\n", se.StoreInterval)
	// fmt.Printf("FileStoragePath = %v\n", se.FileStoragePath)
	// fmt.Printf("Restore = %v\n", se.Restore)
	// fmt.Printf("Adress = %v\n", se.Address)
	flag.Var(&se.paramAddress, "a", "Net address host:port")
	flag.Uint64Var(&se.paramStoreInterval, "i", 300, "storeInterval")
	flag.StringVar(&se.paramFileStoragePath, "f", "save_file", "fileStoragePath")
	flag.StringVar(&se.paramDatabaseDSN, "d", "", "fileStoragePath")
	flag.StringVar(&se.paramKey, "k", "", "key for hash")
	flag.BoolVar(&se.paramRestore, "r", false, "paramRestore")
	flag.StringVar(&se.paramAuditFile, "audit-file", "", "path to audit file")
	flag.StringVar(&se.paramAuditURL, "audit-url", "", "URL for audit logs")

	// fmt.Println("======AFTER PARAMS PARSE-----")
	// fmt.Printf("paramStoreInterval = %v\n", se.paramStoreInterval)
	// fmt.Printf("paramFileStoragePath = %v\n", se.paramFileStoragePath)
	// fmt.Printf("paramRestore = %v\n", se.paramRestore)
	// fmt.Printf("paramAdress = %v\n", se.paramAddress)
	// fmt.Printf("StoreInterval = %v\n", se.StoreInterval)
	// fmt.Printf("FileStoragePath = %v\n", se.FileStoragePath)
	// fmt.Printf("Restore = %v\n", se.Restore)
	// fmt.Printf("Adress = %v\n", se.Address)
}

func (se *ServerConfig) Parse() {

	// fmt.Println("======BEFORE ENV PARSE-----")
	// fmt.Printf("paramStoreInterval = %v\n", se.paramStoreInterval)
	// fmt.Printf("paramFileStoragePath = %v\n", se.paramFileStoragePath)
	// fmt.Printf("paramRestore = %v\n", se.paramRestore)
	// fmt.Printf("paramAdress = %v\n", se.paramAddress)
	// fmt.Printf("StoreInterval = %v\n", se.StoreInterval)
	// fmt.Printf("FileStoragePath = %v\n", se.FileStoragePath)
	// fmt.Printf("Restore = %v\n", se.Restore)
	// fmt.Printf("Adress = %v\n", se.Address)

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
	// fmt.Printf("paramAdress = %v\n", se.paramAddress)
	// fmt.Printf("StoreInterval = %v\n", se.StoreInterval)
	// fmt.Printf("FileStoragePath = %v\n", se.FileStoragePath)
	// fmt.Printf("Restore = %v\n", se.Restore)
	// fmt.Printf("Adress = %v\n", se.Address)

	_, ok1 := problemVars["STORE_INTERVAL"]
	_, ok2 := problemVars["StoreInterval"]
	if !ok1 && !ok2 {
		se.StoreInterval = se.envs.StoreInterval
	} else {
		se.StoreInterval = se.paramStoreInterval
	}

	_, ok1 = problemVars["FILE_STORAGE_PATH"]
	_, ok2 = problemVars["FileStoragePath"]
	if !ok1 && !ok2 {
		se.FileStoragePath = se.envs.FileStoragePath
	} else {
		se.FileStoragePath = se.paramFileStoragePath
	}

	_, ok1 = problemVars["RESTORE"]
	_, ok2 = problemVars["Restore"]
	if !ok1 && !ok2 {
		se.Restore = se.envs.Restore
	} else {
		se.Restore = se.paramRestore
	}

	_, ok1 = problemVars["ADDRESS"]
	_, ok2 = problemVars["Address"]
	if !ok1 && !ok2 {
		se.Address = se.envs.Address
	} else {
		se.Address = se.paramAddress
	}

	_, ok1 = problemVars["DATABASE_DSN"]
	_, ok2 = problemVars["DatabaseDSN"]
	if !ok1 && !ok2 {
		se.DatabaseDSN = se.envs.DatabaseDSN
	} else {
		se.DatabaseDSN = se.paramDatabaseDSN
	}

	_, ok1 = problemVars["KEY"]
	_, ok2 = problemVars["Key"]
	if !ok1 && !ok2 {
		se.Key = se.envs.Key
	} else {
		se.Key = se.paramKey
	}

	_, ok1 = problemVars["AUDIT_FILE"]
	_, ok2 = problemVars["AuditFile"]
	if !ok1 && !ok2 {
		se.AuditFile = se.envs.AuditFile
	} else {
		se.AuditFile = se.paramAuditFile
	}

	_, ok1 = problemVars["AUDIT_URL"]
	_, ok2 = problemVars["AuditURL"]
	if !ok1 && !ok2 {
		se.AuditURL = se.envs.AuditURL
	} else {
		se.AuditURL = se.paramAuditURL
	}
	// fmt.Println("======RESULT-----")
	// fmt.Printf("paramStoreInterval = %v\n", se.paramStoreInterval)
	// fmt.Printf("paramFileStoragePath = %v\n", se.paramFileStoragePath)
	// fmt.Printf("paramRestore = %v\n", se.paramRestore)
	// fmt.Printf("paramAdress = %v\n", se.paramAddress)
	// fmt.Printf("StoreInterval = %v\n", se.StoreInterval)
	// fmt.Printf("FileStoragePath = %v\n", se.FileStoragePath)
	// fmt.Printf("Restore = %v\n", se.Restore)
	// fmt.Printf("Adress = %v\n", se.Address)
}
