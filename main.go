package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var (
	session   mgo.Session
	db        *mgo.Database
	c         *mgo.Collection
	fs        *mgo.GridFS
	startTime time.Time
	root      string
	args      []string
)

type Source struct {
	Id        bson.ObjectId     `bson:"_id"`
	ResultIds []bson.ObjectId   `bson:"results"`
	Files     map[string]string `bson:"files"`
	Arch      string            `bson:"arch"`
	Args      string            `bson:"args,omitempty"`
}

func init() {
	startTime = time.Now()
	root, _ = os.Getwd()
	args = os.Args[1:]

	session, err := mgo.Dial("localhost")
	if err != nil {
		fmt.Println(err)
		return
	}
	db = session.DB("bcc-test")
	c = db.C("sources")
	fs = db.GridFS("fs")
}

func addToCache(s *Source) {
	fmt.Println("Result not found in cache. Running gcc (may take a while)...")

	var gccOut, gccErr bytes.Buffer
	gccCmd := exec.Command("gcc", args...)
	gccCmd.Stderr = &gccErr
	gccCmd.Stdout = &gccOut

	err := gccCmd.Run()
	fmt.Println(gccOut.String())
	if err != nil {
		fmt.Println("Running gcc failed:", err)
		fmt.Println(gccErr.String())
		return
	}
	fmt.Println("gcc has finished. Adding result in cache.")

	sourceId := bson.NewObjectId()
	s.Id = sourceId

	var files []bson.ObjectId
	// insert files that have been created since start
	if err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		relPath, _ := filepath.Rel(root, path)
		// ignore hidden directories and .
		if relPath == "." || strings.Contains(relPath, "/.") {
			return nil
		}

		if info.ModTime().Unix() >= startTime.Unix() { // .After isn't granular enough
			fmt.Println("- " + relPath + " has been generated by gcc and is being added to cache.")
			file, err := fs.Create(sourceId.Hex() + ":" + relPath)
			if err != nil {
				fmt.Println("Creating file failed. Please make sure this program has appropriate permissions.")
				return err
			}
			content, err := os.Open(path)
			if err != nil {
				fmt.Println("Reading file " + path + " failed. Please make sure this program has appropriate permissions.")
				return err
			}
			defer content.Close()

			if _, err = io.Copy(file, content); err != nil {
				fmt.Println("Copying file failed. Please make sure this program has appropriate permissions.")
				return err
			}
			if err = file.Close(); err != nil {
				fmt.Println("Closing file failed. Please make sure this program has appropriate permissions.")
				return err
			}
			files = append(files, file.Id().(bson.ObjectId))
		}
		return nil
	}); err != nil {
		fmt.Println("Failed to walk Directory. Please make sure this program has appropriate permissions.")
		fmt.Println(err)
		return
	}

	// insert
	s.ResultIds = files
	if err = c.Insert(s); err != nil {
		fmt.Println("Error inserting document into database. Please make sure you are connected to the internet.")
		fmt.Println(err)
	}
	return
}

func writeFromCache(s *Source) {
	fmt.Println("Writing files from cache instead of running gcc...")
	resultId := s.Id.Hex()
	targetResults := []bson.M{}
	if err := fs.Find(bson.M{"_id": bson.M{"$in": s.ResultIds}}).All(&targetResults); err != nil {
		fmt.Println("Unable to find cached files with ids", s.ResultIds, ". Please contact support@bowery.io.")
		fmt.Println(err)
		return
	}

	for _, f := range targetResults {
		file, err := fs.OpenId(f["_id"])
		if err != nil {
			// TODO (thebyrd) remove id from cache and handle this gracefully.
			fmt.Println("Unable to find cached file with id ", f["_id"], ". Please contact support@bowery.io.")
			fmt.Println(err)
			return
		}

		outPath := strings.Replace(f["filename"].(string), resultId+":", "", -1)
		outfile, err := os.Create(outPath)
		if err != nil {
			fmt.Println("Failed to create file. Please make sure this program has appropriate permission.")
			fmt.Println(err)
			return
		}

		if _, err = io.Copy(outfile, file); err != nil {
			fmt.Println("Failed to copy file from cache to your computer. Please make sure this program has appropriate permission.")
			fmt.Println(err)
			return
		}

		if err = file.Close(); err != nil {
			fmt.Println("Failed to close resulting file. Please make sure this program has appropriate permission.")
			fmt.Println(err)
			return
		}

		if err = exec.Command("chmod", "+x", outPath).Run(); err != nil {
			fmt.Println("Failed to make resulting file executable. Please make sure this program has appropriate permission.")
			fmt.Println(err)
			return
		}
		fmt.Println("Finished writing " + outPath + ".")
	}
}

func main() {
	defer session.Close()

	s := &Source{
		Arch:  runtime.GOOS + "-" + runtime.GOARCH,
		Args:  strings.Join(args, " "),
		Files: map[string]string{},
	}

	if err := filepath.Walk(root, func(path string, f os.FileInfo, err error) error {
		relPath, _ := filepath.Rel(root, path)
		// ignore hidden directories and .
		if relPath == "." || strings.Contains(relPath, "/.") {
			return nil
		}

		content, _ := ioutil.ReadFile(path)
		s.Files[strings.Replace(relPath, ".", "_", -1)] = fmt.Sprintf("%x", md5.Sum(content))
		return nil
	}); err != nil {
		fmt.Println("Failed:", err)
		return
	}

	err := c.Find(bson.M{"files": s.Files, "arch": s.Arch, "args": s.Args}).One(&s)

	if err != nil && err.Error() == "not found" {
		addToCache(s)
	} else if err != nil {
		fmt.Println("Error connecting to database. Please make sure you are connected to the internet and try again.")
		fmt.Println(err)
	} else {
		writeFromCache(s)
	}
}
