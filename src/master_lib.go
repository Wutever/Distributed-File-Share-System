package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"sort"
	"strconv"
	"strings"
)

var Leader int = 1

var FileMap map[string][]int = make(map[string][]int)
var NodeMap map[int][]string = make(map[int][]string)

type replicaInfo struct {
	prime, secondary, ternary int
}

func readFileMapping(fm map[string][]int, nm map[int][]string) {
	data, err := ioutil.ReadFile("filemappingtable.log")
	//if empty file (init)
	if err != nil {
		return
	}
	b := bytes.NewBuffer(data)
	d := gob.NewDecoder(b)

	// Decoding the serialized data
	//fm := map[string][]int
	err = d.Decode(&fm)
	if err != nil {
		panic(err)
	}
	for k, v := range fm {
		for machineNr := range v {
			nm[machineNr] = append(nm[machineNr], k)
		}
	}
	return
}

func writeFileMapping(fm map[string][]int) int {
	b := new(bytes.Buffer)
	e := gob.NewEncoder(b)
	err := e.Encode(fm)
	if err != nil {
		panic(err)
	}
	ioutil.WriteFile("filemappingtable.log", b.Bytes(), 0644)
	NodeList := GetOnlineMember()
	//keep a backup of the file log
	go put_file_remote(b.Bytes(), "filemappingtable.log", NodeList[1%len(NodeList)]-1)
	go put_file_remote(b.Bytes(), "filemappingtable.log", NodeList[2%len(NodeList)]-1)

	return 0
}

func getMachineNrs(remoteName string, requestMachine int) []int {
	currentLive := GetOnlineMember()
	var currentBackup []int
	hashVal := sha256.Sum256([]byte(remoteName))[0]
	intHashVal := int(hashVal)
	NodeList := GetOnlineMember()
	var machNrs []int
	if len(NodeList) <= 3 {
		primaryBackupIdx := intHashVal % len(NodeList)
		secondaryBackupIdx := (primaryBackupIdx - 1 + len(NodeList)) % len(NodeList)
		ternaryBackupIdx := (primaryBackupIdx + 1 + len(NodeList)) % len(NodeList)
		log.Printf("%d, %d, %d", primaryBackupIdx, secondaryBackupIdx, ternaryBackupIdx)
		machNrs := []int{NodeList[primaryBackupIdx], NodeList[secondaryBackupIdx], NodeList[ternaryBackupIdx]}
		return machNrs
	}

	for {
		newMachineIdx := rand.Intn(len(currentLive))

		if !intInSlice(currentLive[newMachineIdx], currentBackup) && currentLive[newMachineIdx] != requestMachine {
			currentBackup = append(currentBackup, currentLive[newMachineIdx])
		}
		if len(currentBackup) == 3 {
			return currentBackup
		}
	}

	return machNrs
}

//type Master int

func (t *Machine) GetMachineNrsFromMaster(cmd *string, reply *Logs) error {
	//words = command, requesting machine, filename
	words := strings.Split(*cmd, spliter)
	command := words[0]
	requestMachine := words[1]
	filename := words[2]

	//first see if this file is already in the filemap
	mapping, exist := FileMap[filename]
	if exist {
		resultStr := ""
		for idx, ele := range mapping {
			if idx == len(mapping)-1 {
				resultStr = (resultStr + strconv.Itoa(ele))
			} else {
				resultStr = resultStr + strconv.Itoa(ele) + spliter
			}
		}
		if command == "delete" {
			delete(FileMap, filename)
		}
		//resultStr := strings.Join(mapping, spliter)
		reply.Log = resultStr
	} else {
		if command == "delete" || command == "get" {
			resultStr := "0"
			reply.Log = resultStr
		} else {
			//else it's a new file, just get new mappings and do correct bookkeeping
			requestMachNr, _ := strconv.Atoi(requestMachine)
			mapping = getMachineNrs(filename, requestMachNr)
			mapping = append(mapping, requestMachNr)
			FileMap[filename] = mapping
			for _, machine := range mapping {
				NodeMap[machine] = append(NodeMap[machine], filename)
			}
			go writeFileMapping(FileMap)

			resultStr := ""
			for idx, ele := range mapping {
				if idx == len(mapping)-1 {
					resultStr = (resultStr + strconv.Itoa(ele))
				} else {
					resultStr = resultStr + strconv.Itoa(ele) + spliter
				}
				log.Printf("the element: %s", strconv.Itoa(ele))

			}
			log.Printf("%s", resultStr)
			//resultStr := strings.Join(mapping, spliter)
			reply.Log = resultStr
		}
	}

	//reply.prime = machineNrs[0]
	//reply.secondary = machineNrs[1]
	//reply.ternary = machineNrs[2]
	return nil
}

// dispatch a file to corresponding machines
//func dispatchFiles(localName string, remoteName string) int {
//	machineNrs := getMachineNrs(remoteName)
//	for _, maNr := range machineNrs{
//		go put_file(localName, remoteName, maNr)
//	}
//	return 0
//}

func CheckStatus(selfId int) {
	if GetNeedAction() {
		log.Print("!!!!!!!!!!!!!!!!!!!!!!!sssss")
		NewLeader := electLeader(NodeFailed)
		if NewLeader != Leader && NewLeader == selfId {
			readFileMapping(FileMap, NodeMap)
		}
		Leader = NewLeader
		failRescue(NodeFailed)
		SetNeedAction(false)
	}
}

// running bully algorithm
func electLeader(failId int) int {
	NodeList := GetOnlineMember()
	return MinIntSlice(NodeList)
}

func failRescue(failId int) {

	currentLive := GetOnlineMember()
	filesInFailNode := NodeMap[failId]
	log.Printf("theNodeMap is %v,the failId is %d", filesInFailNode, failId)
	//loop over all the files in this fail node
	for _, file := range filesInFailNode {
		currentBackup := FileMap[file]
		log.Printf("%v", currentBackup)
		newMachineIdx := 0
		for {
			newMachineIdx = rand.Intn(len(currentLive))
			if !intInSlice(currentLive[newMachineIdx], currentBackup) || len(currentLive) <= 2 {
				break
			}
		}
		//found a new machine to put file!
		//TODO: remote put file!!!!!!
		//put_file(file, file, currentLive[newMachineIdx])
		liveCopyNode := 0
		for _, node := range currentBackup {
			if node != failId {
				liveCopyNode = node
				break
			}
		}
		//wg := sync.WaitGroup{}
		//wg.Add(1)
		command := "putM" + spliter + file + spliter + strconv.Itoa(currentLive[newMachineIdx]-1)
		machineInfo := machines[liveCopyNode-1].machine_addrs + ":" + machines[liveCopyNode-1].machine_port
		log.Printf("the info is %s", machineInfo)
		go func() {
			// defer func() {
			// 	recover()
			// 	wg.Done()
			// }()
			client_socket, err := jsonrpc.Dial("tcp", machineInfo)
			if err != nil {
				log.Printf("error in file rescue")
			}
			var reply_logs Logs
			client_socket.Call("Machine.Get_local_log", &command, &reply_logs)
			//fmt.Printf("remote call success")
			if err != nil {
				fmt.Printf("remote call error")
			}
			client_socket.Close()
		}()
		//wg.Wait()
		temp := FileMap[file]
		var tempint []int
		for _, n := range temp {
			if n != failId {
				tempint = append(tempint, n)
			}
		}
		tempint = append(tempint, currentLive[newMachineIdx])
		FileMap[file] = tempint
		fmt.Printf("the new fileMap is %v", FileMap[file])
	}
}

func ServeAsMaster() {
	machine := new(Machine)
	rpc.Register(machine)
	tcpAddr, _ := net.ResolveTCPAddr("tcp", ":12543")
	listener, _ := net.ListenTCP("tcp", tcpAddr)
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		jsonrpc.ServeConn(conn)
	}
}

//helper functions
func MinIntSlice(v []int) int {
	sort.Ints(v)
	return v[0]
}

func intInSlice(a int, list []int) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
