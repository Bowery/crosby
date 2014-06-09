package main

import (
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
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
	dbHost    string
	apiHost   string
	homeVar   string
	wg        sync.WaitGroup
)

type Session struct {
	User   User   `json:"user,omitempty"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type User struct {
	ID          string    `json:"id"`
	Name        string    `json:"name,omitempty"`
	Email       string    `json:"email,omitempty"`
	Password    string    `json:"password,omitempty"`
	Salt        string    `json:"salt,omitempty"`
	StripeToken string    `json:"stripeToken,omitempty"`
	Expiration  time.Time `json:"expiration,omitempty"`
}

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
	dbHost = "ec2-54-83-94-130.compute-1.amazonaws.com"
	apiHost = "crosby.io"

	if os.Getenv("ENV") == "development" {
		dbHost = "localhost"
		apiHost = "localhost:3000"
	}

	session, err := mgo.Dial(dbHost)
	if err != nil {
		fmt.Println(err)
		panic("could not connect to crosby")
		return
	}
	db = session.DB("crosby")
	c = db.C("sources")
	fs = db.GridFS("fs")

	homeVar = "HOME"
	if runtime.GOOS == "windows" {
		homeVar = "USERPROFILE"
	}
}

func CurrentUser() (*User, error) {
	user := User{}
	configPath := filepath.Join(os.Getenv(homeVar), ".crosbyconf")

	// ghetto version of "touch"
	file, err := os.OpenFile(configPath, os.O_RDONLY|os.O_CREATE|os.O_APPEND, 0664)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if err = gob.NewDecoder(file).Decode(&user); err != nil {
		return nil, err
	}
	return &user, nil
}

//
// Parses name and email and creates a new user.
// Will also check server to see if the user exists and get expiration time.
//
func CreateUser() (*User, error) {
	// Get name and email from git if possible
	// generate id
	name, _ := exec.Command("git", "config", "user.name").Output()
	email, _ := exec.Command("git", "config", "user.email").Output()

	var strName string
	if len(name) > 0 {
		strName = string(name[:len(name)-1])
	}

	var strEmail string
	if len(email) > 0 {
		strEmail = string(email[:len(email)-1])
	}

	u := &User{
		Name:  strName,
		Email: strEmail,
	}

	res, err := http.PostForm("http://"+apiHost+"/session",
		url.Values{
			"name":  {u.Name},
			"email": {u.Email},
		})
	if err != nil {
		return u, err
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return u, err
	}

	s := &Session{}
	if err := json.Unmarshal(body, s); err != nil {
		return u, err
	}

	if s.Status == "failed" {
		return u, errors.New(s.Error)
	}

	// So that the server assigned ID is persisted
	u = &s.User

	var raw bytes.Buffer
	if err := gob.NewEncoder(&raw).Encode(u); err != nil {
		return u, err
	}

	if err := ioutil.WriteFile(filepath.Join(os.Getenv(homeVar), ".crosbyconf"), raw.Bytes(), os.ModePerm); err != nil {
		return u, err
	}

	return u, nil
}

//
// If the session has expired
//
func ValidateSession() error {
	if os.Getenv("ENV") == "development" {
		return nil
	}

	user, err := CurrentUser()
	if err == io.EOF {
		user, err = CreateUser()
	}

	if err != nil {
		return err
	}

	res, err := http.Get("http://" + apiHost + "/session/" + user.ID)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	s := &Session{}
	if err := json.Unmarshal(body, s); err != nil {
		return err
	}

	if s.Status == "failed" {
		return errors.New(s.Error)
	}

	if s.Status == "expired" {
		fmt.Println("Hi", s.User.Name, "!")
		fmt.Println("Your free trial has expired. Please register at http://crosby.io/signup")
		fmt.Println("Your Account Number is", s.User.ID)
		return errors.New("You must register to continue using Crosby.")
	}

	// Update config file incase something has changed server side
	go func() {
		if s.Status == "found" {
			var raw bytes.Buffer
			gob.NewEncoder(&raw).Encode(s.User)
			ioutil.WriteFile(filepath.Join(os.Getenv(homeVar), ".crosbyconf"), raw.Bytes(), os.ModePerm)
		}
	}()
	return nil
}

func AddToCache(s *Source) {
	fmt.Println("Result not found in cache. Running command (may take a while)...")

	var cmdOut, cmdErr bytes.Buffer
	fmt.Println(args[0], args[1:])
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stderr = &cmdErr
	cmd.Stdout = &cmdOut

	err := cmd.Run()
	fmt.Println(cmdOut.String())
	if err != nil {
		fmt.Println("Running command failed:", err)
		fmt.Println(cmdErr.String())
		return
	}
	fmt.Println("Command has finished. Adding result in cache.")

	sourceId := bson.NewObjectId()
	s.Id = sourceId

	var files []bson.ObjectId
	// insert files that have been created since start
	if err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		relPath, _ := filepath.Rel(root, path)
		// ignore hidden directories and .
		if relPath == "." || strings.Contains(relPath, "/.") || info.IsDir() {
			return nil
		}

		if info.ModTime().Unix() >= startTime.Unix() { // .After isn't granular enough
			fmt.Println("- " + relPath + " has been generated by command and is being added to cache.")
			file, err := fs.Create(sourceId.Hex() + ":" + relPath)
			if err != nil {
				return err
			}
			content, err := os.Open(path)
			if err != nil {
				return err
			}
			defer content.Close()

			if _, err = io.Copy(file, content); err != nil {
				return err
			}
			if err = file.Close(); err != nil {
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

func WriteFromCache(s *Source) {
	fmt.Println("Writing files from cache instead of running command...")
	resultId := s.Id.Hex()
	targetResults := []bson.M{}
	if err := fs.Find(bson.M{"_id": bson.M{"$in": s.ResultIds}}).All(&targetResults); err != nil {
		fmt.Println("Unable to find cached files with ids", s.ResultIds, ". Please contact support@bowery.io.")
		fmt.Println(err)
		return
	}

	for _, f := range targetResults {
		wg.Add(1)
		go writeFile(f, resultId)
	}
	wg.Wait()
}

func writeFile(f map[string]interface{}, resultId string) {
	defer wg.Done()

	file, err := fs.OpenId(f["_id"])
	if err != nil {
		// TODO (thebyrd) remove id from cache and handle this gracefully.
		fmt.Println("Unable to find cached file with id ", f["_id"], ". Please contact support@bowery.io.")
		fmt.Println(err)
		return
	}

	outPath := strings.Replace(f["filename"].(string), resultId+":", "", -1)
	if err = os.MkdirAll(filepath.Dir(outPath), os.ModePerm|os.ModeDir); err != nil {
		fmt.Println(err)
		return
	}

	outfile, err := os.Create(outPath)
	if err != nil {
		fmt.Println("Failed to create file. Please make sure this program has appropriate permission.")
		fmt.Println(err)
		return
	}
	defer outfile.Close()

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

func main() {
	defer session.Close()

	if len(os.Args) <= 1 {
		fmt.Println("Error: Must Specify Command to Run")
		fmt.Println("Usage: crosby <command>")
		return
	}

	if err := ValidateSession(); err != nil {
		fmt.Println(err)
		return
	}

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

	query := bson.M{}
	for key := range s.Files {
		query["files."+key] = s.Files[key]
	}
	query["arch"] = s.Arch
	query["args"] = s.Args

	results := []Source{}
	err := c.Find(query).All(&results)
	notFound := err != nil || len(results) == 0

	for _, result := range results {
		if len(result.Files) == len(s.Files) {
			s = &result
			notFound = false
			break
		}
	}

	if notFound {
		AddToCache(s)
	} else if err == nil {
		WriteFromCache(s)
	} else {
		fmt.Println("Error connecting to database. Please make sure you are connected to the internet and try again.")
		fmt.Println(err)
	}
}
