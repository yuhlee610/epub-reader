package main

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

func loadLocalEnv() error {
	return loadDotEnv(".env")
}

func loadDotEnv(path string) error {
	if err := godotenv.Load(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return nil
}
