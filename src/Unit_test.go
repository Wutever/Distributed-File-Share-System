package main

import (
	"failure"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestJoinIntroducer(t *testing.T) {
	var text = "put a b\n"
	if text == "leave\n" {
		fmt.Println("Leaving group")
		failure.LeaveGroup()
	} else {
		text = strings.Replace(text, "\n", "", -1)
		words := strings.Fields(text)

		if words[0] == "put" {
			for _, element := range words {
				fmt.Printf("%s", element)
			}
			fmt.Println("put command detect")
		}

		if words[0] == "get" {
			fmt.Println("get command detect")
		}

	}
}
func TestFileTransfer(t *testing.T) {
	x := new(Logs)
	spliter := "themagicstring"

	a := "thetile"
	b := "thecontent"
	tranmist := a + spliter + b

	words := strings.Split(tranmist, spliter)
	err := ioutil.WriteFile(words[0], []byte(words[1]), 0644)
	if err != nil {
		x.Log = "fail!"
	} else {
		x.Log = "sucess!"
	}
}
func TestFileSystemInitialize(t *testing.T) {
	Working_dirctory, err = filepath.Abs(filepath.Dir(os.Args[0]))
	fmt.Printf("the Working_dirctory: %s", Working_dirctory)
	if err != nil {
		log.Fatal(err)
	}
	sdfsDir := path.Join(Working_dirctory, Sdfs_folder)
	fmt.Printf("the joint dirctory : %s", sdfsDir)
	if _, err := os.Stat(sdfsDir); os.IsNotExist(err) {
		os.Mkdir(sdfsDir, 0700)
	}
	if _, err := os.Stat(sdfsDir); os.IsNotExist(err) {
		fmt.Println("creation failed") // path/to/whatever does not exist""
	}
}

func TestGetRecentFile(t *testing.T) {
	dir := "SDFS/www"
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var modTime time.Time
	var names []string
	for _, fi := range files {
		if fi.Mode().IsRegular() {
			if !fi.ModTime().Before(modTime) {
				if fi.ModTime().After(modTime) {
					modTime = fi.ModTime()
					names = names[:0]
				}
				names = append(names, fi.Name())
			}
		}
	}
	if len(names) > 0 {
		fmt.Println(modTime, names)
	}
	fmt.Println(names[0])

}

func TestQueryHostName(t *testing.T) {
	stringa, _ := os.Hostname()
	fmt.Println(stringa)
	stringa = stringa[15:17]
	fmt.Println(stringa)
}

func TestDeleteVersionFile(t *testing.T) {
	Working_dirctory, err = filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
		return
	}
	Working_dirctory = path.Join("_/home/jzhu62/ece428_mp/MP3/src/", Sdfs_folder)
	fmt.Printf("the workingdir is %s\n", Working_dirctory)
	if _, err := os.Stat(Working_dirctory); os.IsNotExist(err) {
		os.Mkdir(Working_dirctory, 0700)
	} else {
		log.Println("oops, did not create working sdfs directory")
	}
	dir, err := ioutil.ReadDir(Sdfs_folder + "/" + "www")
	if err != nil {
		log.Printf("the file %s does not exist!", "www")
	}
	for _, d := range dir {
		stuff := os.Remove(path.Join(Working_dirctory, "www", d.Name()))
		if stuff == nil {
			log.Printf("%s", d.Name())
		}
	}
	// stuff := os.RemoveAll(Sdfs_folder + "/" + "www")
	// if stuff == nil {
	// 	log.Println("path not exist")
	// }
	os.Remove(path.Join(Working_dirctory, "www"))
}
