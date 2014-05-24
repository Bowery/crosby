package main

import (
	"crypto/md5"
	"fmt"
	"io/ioutil"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	query := bson.M{}

	root, _ := os.Getwd()

	if err := filepath.Walk(root, func(path string, f os.FileInfo, err error) error {
		relPath, _ := filepath.Rel(root, path)
		// ignore hidden directories and .
		if relPath == "." || strings.Contains(relPath, "/.") {
			return nil
		}

		content, _ := ioutil.ReadFile(path)
		query[relPath] = fmt.Sprintf("%x", md5.Sum(content))
		return nil
	}); err != nil {
		fmt.Println("Failed:", err)
	}

	session, err := mgo.Dial("localhost")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer session.Close()

	c := session.DB("bcc-test").C("sources")

	result := map[string]string{}
	err = c.Find(query).One(&result)
	if err.Error() == "not found" {
		fmt.Println("Not Found. Should run gcc and then insert.")
		if err = c.Insert(query); err != nil {
			fmt.Println(err)
		}
		return
	} else if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Found!")
	fmt.Println(result)
}
