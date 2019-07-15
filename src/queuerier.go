package main

import (
	"net/rpc/jsonrpc"
	"strings"
	"time"

	//"bufio"
	"fmt"
	"log"

	//"net/rpc"
	//"os"
	//"strings"
	"sync"
)

const master = 0
const machine = 1

var NR_OF_MACHINE = 10

//type Args struct {
//	exeCmd int
//}

type fileclient struct {
	machine_addrs string
	machine_port  string
}

var machines = []fileclient{
	fileclient{"fa18-cs425-g22-01.cs.illinois.edu", "2345"},
	fileclient{"fa18-cs425-g22-02.cs.illinois.edu", "2345"},
	fileclient{"fa18-cs425-g22-03.cs.illinois.edu", "2345"},
	fileclient{"fa18-cs425-g22-04.cs.illinois.edu", "2345"},
	fileclient{"fa18-cs425-g22-05.cs.illinois.edu", "2345"},
	fileclient{"fa18-cs425-g22-06.cs.illinois.edu", "2345"},
	fileclient{"fa18-cs425-g22-07.cs.illinois.edu", "2345"},
	fileclient{"fa18-cs425-g22-08.cs.illinois.edu", "2345"},
	fileclient{"fa18-cs425-g22-09.cs.illinois.edu", "2345"},
	fileclient{"fa18-cs425-g22-10.cs.illinois.edu", "2345"},
}
var err error

func something(text string, index int, masterorhost int) string {
	wg := sync.WaitGroup{}
	c := make(chan int, 10)
	start := time.Now()
	wg.Add(1)
	reply := DialRemoteMachine(machines[index].machine_addrs+":"+machines[index].machine_port, text, &wg, index, c, masterorhost)
	wg.Wait()
	elapsed := time.Since(start)
	log.Printf("\n***********\nTime used: %s\n***********\n", elapsed)
	return reply

}

func DialRemoteMachine(machineInfo string, commandToRun string, wg *sync.WaitGroup, machineNr int, c chan int, masterorhost int) string {
	defer func() {
		recover()
		wg.Done()
	}()
	//log.Print(commandToRun + "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	//log.Print(machineInfo)
	client_socket, err := jsonrpc.Dial("tcp", machineInfo)

	if err != nil {
		a := "Machine " + machineInfo + " is offline! "
		log.Printf("\n Machine %s is offline! \n", machineInfo)
		return a
	}
	var reply_logs Logs
	if masterorhost == machine {
		client_socket.Call("Machine.Get_local_log", &commandToRun, &reply_logs)
	} else {
		log.Printf("calling master")
		client_socket.Call("Machine.GetMachineNrsFromMaster", &commandToRun, &reply_logs)
		log.Printf("the replay meesage from maaster is %s", reply_logs.Log)
	}

	//fmt.Printf("remote call success")
	if err != nil {
		fmt.Printf("remote call error")
	}
	//log.Printf("\n[Machine %d Log:]\n ", machineNr)
	c <- strings.Count(reply_logs.Log, "\n")
	client_socket.Close()
	//wg.Done()
	return reply_logs.Log
}
