package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"net/rpc/jsonrpc"
	"strconv"
	"strings"
	"unsafe"
)

type Logs struct {
	Log        string
	Machine_nr int
}
type Machine int

func (t *Machine) Get_local_log(args *string, reply *Logs) error {

	words := strings.Split(*args, spliter)
	fmt.Printf("received arguments! name : %s \n", words[0])
	if words[0] == "put" {
		log.Printf("the size is %d", unsafe.Sizeof(words[2]))
		//writelocalfile(words[1], words[2])
		sdfs_writelocalfile(words[1], words[2])
		return nil
	} else if words[0] == "get" {
		lv, _ := strconv.Atoi(words[1])
		dat, localname := sdfs_readlocalfile(words[2], lv)
		if dat == nil {
			reply.Log = filenotfound
			log.Println("unable to find the file to push to remote FS")
		} else {
			reply.Log = localname + spliter + string(dat)
		}

	} else if words[0] == "putM" {
		index, err := strconv.Atoi(words[2])
		log.Printf("the putM destinaiton is %d", index)
		if err != nil {
			log.Println("failed to convert to integer while doing PutM command")
		}
		a, _ := sdfs_readlocalfile(words[1], 1)
		put_file_remote(a, words[1], index)
	} else if words[0] == "delete" {
		sdfs_deletelocalfile(words[1])
	}
	return nil
}

func openfileport() {
	machine := new(Machine)
	rpc.Register(machine)

	tcpAddr, _ := net.ResolveTCPAddr("tcp", ":2345")
	listener, _ := net.ListenTCP("tcp", tcpAddr)
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		jsonrpc.ServeConn(conn)

	}

}
