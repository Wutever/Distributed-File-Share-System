package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

var Working_dirctory = ""
var Working_dirctory_src = ""
var hostname = 0

const spliter = "themagicstring"
const filespliter = "thefilestring"
const filenotfound = " filenotfound"
const Sdfs_folder = "SDFS"
const unabletodelete = "unabledelete"

type client struct {
	machine_addrs string
	machine_port  string
}

func fs_initialize() {
	hostname = GetHostindex()
	Working_dirctory, err = filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
		return
	}
	Working_dirctory_src = Working_dirctory
	Working_dirctory = path.Join(Working_dirctory, Sdfs_folder)

	if _, err := os.Stat(Working_dirctory); os.IsNotExist(err) {
		os.Mkdir(Working_dirctory, 0700)
	} else {
		log.Println("oops, did not create working sdfs directory")
	}

}
func get_versions_file(remotefilename string, localfilename string, masterindex int, lv string) {
	index, _ := strconv.Atoi(lv)
	log.Print("staart gettting file")
	for i := index; i > 0; i-- {
		log.Printf("this is the %d iteration", i)
		stringindex := strconv.Itoa(i)
		remoteversion, remotecontent := get_File(remotefilename, localfilename, masterindex, stringindex)
		log.Printf("the remote version aquired, version is %d,\n ", remoteversion)
		if index == 1 {
			writelocalfile(localfilename, remotecontent)
		} else {
			writelocalfile(localfilename+strconv.Itoa(remoteversion), remotecontent)
		}

	}

}
func get_version_file_path(filename string, lv int) string {
	Working_file_dirctory := path.Join(Working_dirctory, filename)
	if _, err := os.Stat(Working_file_dirctory); os.IsNotExist(err) {
		log.Println("file does not   exists")
		return filenotfound
	}
	dir := Sdfs_folder + "/" + filename
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var names []string
	for i := 0; i < len(files)-1; i++ {
		for j := 0; j < len(files)-i+-1; j++ {
			if files[j].ModTime().Before(files[j+1].ModTime()) {
				temp := files[j]
				files[j] = files[j+1]
				files[j+1] = temp
			}
		}
	}
	// for _, fi := range files {
	// 	if fi.Mode().IsRegular() {
	// 		if !fi.ModTime().Before(modTime) {
	// 			if fi.ModTime().After(modTime) {
	// 				modTime = fi.ModTime()
	// 				names = names[:0]
	// 			}
	// 			names = append(names, fi.Name())
	// 		}
	// 	}
	// }
	// if len(names) > 0 {
	// 	fmt.Println(modTime, names)
	// }
	if len(files) < lv {
		log.Printf("there is not enough version to offer! we have %d versions, want %d versions  ", len(names), lv)
		return path.Join(Working_file_dirctory, files[len(files)-1].Name())
	}
	return path.Join(Working_file_dirctory, files[lv-1].Name())
}
func write_version_file_path(filename string) string {
	Working_file_dirctory := path.Join(Working_dirctory, filename)
	if _, err := os.Stat(Working_file_dirctory); os.IsNotExist(err) {
		os.Mkdir(Working_file_dirctory, 0700)
	} else {
		log.Println("older version exists")
	}
	now := time.Now()
	sec := now.Unix()
	return path.Join(Working_file_dirctory, filename+filespliter+strconv.FormatInt(sec, 10))
}

func delete_version_file(filename string) {
	dir, err := ioutil.ReadDir(path.Join(Sdfs_folder, filename))
	if err != nil {
		log.Printf("the file %s does not exist!", filename)
	}
	for _, d := range dir {
		stuff := os.Remove(path.Join(Working_dirctory, filename, d.Name()))
		if stuff == nil {
			log.Printf("%s", d.Name())
		}
	}
	os.Remove(path.Join(Working_dirctory, filename))
}
func sdfs_deletelocalfile(fileName string) {
	delete_version_file(fileName)
}
func sdfs_readlocalfile(fileName string, lv int) ([]byte, string) {
	read_path := get_version_file_path(fileName, lv)
	dat, err := ioutil.ReadFile(read_path)
	if err != nil {
		log.Println("read failed")
		return nil, filenotfound
	} else {
		return dat, filepath.Base(read_path)
	}
}

func sdfs_writelocalfile(fileName string, data string) {
	write_path := write_version_file_path(fileName)
	err := ioutil.WriteFile(write_path, []byte(data), 0644)
	if err != nil {
		log.Print("write local  file fail!\n")
	} else {
		log.Print("write local success !\n")
	}
}
func writelocalfile(fileName string, data string) {
	write_path := path.Join(Working_dirctory_src, fileName)
	err := ioutil.WriteFile(write_path, []byte(data), 0644)
	if err != nil {
		log.Print("write local  file fail!\n")
	} else {
		log.Print("write local successaaaa !\n")
	}
}
func readlocalfile(fileName string) []byte {
	read_path := path.Join(Working_dirctory_src, fileName)
	dat, err := ioutil.ReadFile(read_path)
	if err != nil {
		log.Println("read failed")
		return nil
	} else {
		return dat
	}

}
func ls_File(remotename string, index int) {

	reply := something("get"+spliter+strconv.Itoa(hostname+1)+spliter+remotename, index, master)
	if reply == "0" {
		log.Println("no file in the file system match the name typed!")
	}
	log.Printf("current machine that store file :%s ", remotename)
	machines := strings.Split(reply, spliter)
	for _, element := range machines {
		if element == "" {
			continue
		}
		log.Printf("machine : %s", element)
	}
}
func get_File(remotename string, localname string, index int, lv string) (int, string) {
	reply := something("get"+spliter+strconv.Itoa(hostname+1)+spliter+remotename, index, master)
	if reply == "0" {
		log.Println(" no files realted found on the file system!")
	}
	machines := strings.Split(reply, spliter)
	var names []string
	var contents []string
	for _, element := range machines {
		if element == "" {
			continue
		}
		log.Printf("target getting index received, value:%s", element)
		indexs, _ := strconv.Atoi(element)
		reply = something("get"+spliter+lv+spliter+remotename, indexs-1, machine)
		if reply == filenotfound {
			log.Printf("file not found at index %d", indexs)
			continue
		}
		replymsg := strings.Split(reply, spliter)
		names = append(names, replymsg[0])
		_ = names
		contents = append(contents, replymsg[1])
		_ = contents
	}
	var versionarray []int
	maxindex := 0
	maxversion := 0
	for index, element := range names {
		words := strings.Split(element, filespliter)
		fileversion := words[1]
		fileversionInt, _ := strconv.Atoi(fileversion)
		if fileversionInt >= maxversion {
			maxindex = index
		}
		versionarray = append(versionarray, fileversionInt)
	}

	return versionarray[maxindex], contents[maxindex]

	//now we only get the last one
	// if reply == filenotfound {
	// 	log.Println("remote file not found")
	// } else {
	// 	log.Printf("the remote file succesfully aquired,remote name is %s \nfile content is %s ", remotename, reply)
	// 	writelocalfile(localname, reply)
	// }

}

func put_File(localname string, remotename string, index int) {
	//dat := sdfs_readlocalfile(localname)
	dat := readlocalfile(localname)
	if dat == nil {
		log.Println("local file read failed ,can not put in file")
		return
	}
	log.Printf("the local file ready to put to remote!,name is %s\n", localname)
	reply := something("put"+spliter+strconv.Itoa(hostname+1)+spliter+remotename, index, master)
	log.Printf(" the received messge is %s", reply)
	machines := strings.Split(reply, spliter)
	for _, element := range machines {
		log.Printf("target machineindex received, value:%s", element)
		if element == "" {
			continue
		}
		indexs, _ := strconv.Atoi(element)
		put_file_remote(dat, remotename, indexs-1)
	}
	// writelocalfile("bintest1", string(dat))
	// a := "put" + spliter + localname + spliter + string(dat)
	// words := strings.Split(a, spliter)
	// writelocalfile("bintest2", string(words[2]))

	//sdfs_writelocalfile(remotename, string(dat))

}
func put_file_remote(data []byte, name string, index int) {
	datas := "put" + spliter + name + spliter + string(data)
	log.Printf("the of pre-put  size is %d", unsafe.Sizeof(datas))
	something(datas, index, machine)

}

func delete_File(remotename string, index int) {
	reply := something("delete"+spliter+strconv.Itoa(hostname+1)+spliter+remotename, index, master)
	if reply == "0" {
		log.Println("file in sdfs not found,can't delete")
		return
	}
	deleteIndex := strings.Split(reply, spliter)
	for _, element := range deleteIndex {
		log.Printf("target machineindex received, value:%s", element)
		Intindex, _ := strconv.Atoi(element)
		replymsg := something("delete"+spliter+remotename, Intindex-1, machine)
		if replymsg == filenotfound || replymsg == unabletodelete {
			log.Printf("delete file name : %s in machine %d failed because %s", remotename, Intindex, replymsg)
		}
	}

}
