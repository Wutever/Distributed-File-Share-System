package main

import (
	"bufio"
	"fmt"
	"grep_server"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {

	println("Starting server")
	log.SetFlags(log.Lshortfile)
	go openfileport()
	Initialize()
	fs_initialize()

	//get_version_file_path("version1", 3)
	// go failure.SendPingTest([]string{"fa18-cs425-g27-07.cs.illinois.edu"})

	go func() {
		CheckStatus(GetHostindex() + 1)
	}()
	go func() {
		fmt.Print("Type 'leave' to shut down server\n")
		for {
			fmt.Println("*****************************************************")
			index := Leader - 1
			reader := bufio.NewReader(os.Stdin)
			text, _ := reader.ReadString('\n')
			if text == "leave\n" {
				fmt.Println("Leaving group")
				LeaveGroup()
			} else {
				text = strings.Replace(text, "\n", "", -1)
				words := strings.Fields(text)

				if words[0] == "put" {
					put_File(words[1], words[2], index)
				} else if words[0] == "get" {
					get_versions_file(words[1], words[2], index, "1")
				} else if words[0] == "list" {
					PrintMembership()
				} else if words[0] == "store" {
					files, err := ioutil.ReadDir(Working_dirctory)
					if err != nil {
						log.Fatal(err)
					}

					for _, f := range files {
						fmt.Println(f.Name())
					}
				} else if words[0] == "createTestFile" {
					currentTime := time.Now()
					timeStampString := currentTime.Format("2006-01-02 15:04:05")
					layOut := "2006-01-02 15:04:05"
					timeStamp, _ := time.Parse(layOut, timeStampString)
					hr, min, sec := timeStamp.Clock()
					writelocalfile(strconv.Itoa(hr)+strconv.Itoa(min)+strconv.Itoa(sec), strconv.Itoa(hr)+strconv.Itoa(min)+strconv.Itoa(sec))
				} else if words[0] == "delete" {
					delete_File(words[1], index)
				} else if words[0] == "ls" {
					ls_File(words[1], index)
				} else if words[0] == "get-versions" {
					if words[2] == "0" {
						log.Printf("num of version can't be 0!")
					}
					get_versions_file(words[1], words[3], index, words[2])
				}

			}

		}
	}()

	grep_server.OpenGrepPort()

	// Keep the program running so it doesn't close the port
	for {
	}
}
