package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func getCurrentExecDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return dir
}

func moveFile(src string, dst string) error {

	//This is debug code...
	fmt.Println("Moving from: ", src, " to: ", dst)
	err := os.Rename(src, dst)
	return err
}

func pathExists(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}
	return true

}

func createIfNotExist(path string) error {

	if pathExists(path) {
		fmt.Println("Current Path: ", path, " exist...")
	} else {
		fmt.Println("Current Path: ", path, " does not exist, creating...")
		return os.MkdirAll(path, os.ModePerm)
	}
	return nil
}
