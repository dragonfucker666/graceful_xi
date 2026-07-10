package getenv

import (
	"os"
	"log"
)

func GetEnvOrPanic(name string) string {
	value, ok := os.LookupEnv(name)
	if !ok {
		log.Panicln("Environment variable " + name + " not present")
	}
	return value
}

func GetEnvOrDefault(name string, default_ string) string {
	value, ok := os.LookupEnv(name)
	if !ok {
		value = default_
	}
	return value
}
