package main

import (
	"encoding/json"

	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/fatih/color"

	"shared"
)

var NodeFailed int = 0
var NeedAction bool = false

// Describes a server, may need to take out Hostname to reduce network bandwidth
type ServerInfo struct {
	Number   int
	Hostname string
}

// How individual entries in the membership list are identified
type MembershipId struct {
	TimeStamp time.Time
	ServNum   uint8
	Failed    bool
}
type MembershipEntry struct {
	Id          MembershipId
	Changed     bool
	ReceivedACK bool
	Mutex       sync.Mutex
}

type MembershipList struct {
	Servers       [shared.NumServers + 1]MembershipEntry
	UpdateFT      bool
	UpdateFTMutex sync.Mutex
}

type FingerTable struct {
	Entries [shared.FingerTableSize]ServerInfo
}

var Introducer = ServerInfo{1, shared.GetServerAddress(1)}

var membershipList MembershipList
var fingerTable FingerTable
var fileLog = shared.OpenLogFile(fmt.Sprintf("detectFail%d.log", shared.GetOwnServerNumber()))
var green = color.New(color.FgGreen).SprintFunc()
var red = color.New(color.FgRed).SprintFunc()

const memListBufferSize int = 1024

func (servInf *ServerInfo) Str() string {
	return fmt.Sprintf("ServerID: %2d", servInf.Number)
}
func (mlId *MembershipId) Str() string {
	return fmt.Sprintf("Server Number: %d\nTimestamp: %v\nFailed: %v", mlId.ServNum, mlId.TimeStamp, mlId.Failed)
}
func (mlEntry *MembershipEntry) Str() (ret string) {
	mlEntry.Mutex.Lock()
	ret = fmt.Sprintf("%s\nChanged: %v\nReceivedACK: %v\n", mlEntry.Id.Str(), mlEntry.Changed, mlEntry.ReceivedACK)
	mlEntry.Mutex.Unlock()
	return
}
func (ml *MembershipList) Str(onlyPrintNums bool) (ret string) {
	for _, entry := range ml.Servers {
		if entry.Id.ServNum != 0 {
			if onlyPrintNums {
				if entry.Id.Failed {
					ret += fmt.Sprintf("%s, ", red(entry.Id.ServNum))
				} else {
					ret += fmt.Sprintf("%s, ", green(entry.Id.ServNum))
				}
			} else {
				ret += entry.Str() + "\n"
			}
		}
	}
	if ret == "" {
		ret = "Empty membership list\n"
	}
	return
}
func (ft *FingerTable) Str() (ret string) {
	for _, entry := range ft.Entries {
		if entry.Number != 0 {
			ret += entry.Str() + "\n\n"
		}
	}
	if ret == "" {
		ret = "Empty finger table\n"
	}
	return
}

// This function assumes you already have the lock to the entry
func (mlEntry *MembershipEntry) MarkFailed() {
	mlEntry.Id.Failed = true
	mlEntry.Changed = true

	//rescue the file from the failed node!
	NodeFailed = int(mlEntry.Id.ServNum)
	NeedAction = true
}

func (ml *MembershipList) GetChanged(returnAll bool) (changedML []MembershipId) {
	for i, entry := range ml.Servers {
		entry.Mutex.Lock()
		if entry.Changed == true || (returnAll && entry.Id.ServNum != 0 && !entry.Id.Failed) {
			changedML = append(changedML, entry.Id)
			ml.Servers[i].Changed = false
		}
		entry.Mutex.Unlock()
	}
	return
}

// Update the membership list
func (ml *MembershipList) Update(changedIds []MembershipId) {
	for _, newId := range changedIds {
		servNum := newId.ServNum
		update := false

		ml.Servers[servNum].Mutex.Lock()

		// Update if we don't have the server id
		// Update if new timestamp is later
		// Update if new entry says it failed but we thought it was alive
		// if reflect.DeepEqual(newId.Server, ml.Servers[servNum].Id.Server) == false {
		if newId.ServNum != ml.Servers[servNum].Id.ServNum {
			fileLog.Printf("Server %d has joined the network\n", servNum)
			update = true
		} else if newId.TimeStamp.After(ml.Servers[servNum].Id.TimeStamp) {
			fileLog.Printf("Server %d has rejoined us!\n", servNum)
			update = true
		} else if newId.Failed == true && ml.Servers[servNum].Id.Failed == false {
			if int(newId.ServNum) == shared.GetOwnServerNumber() {
				fileLog.Printf("Everyone thinks I'm a failure\n")
				time.Sleep(500 * time.Millisecond)
				println("Restarting own process (not really right now")
				//Join()
			} else {
				fileLog.Printf("I received a message that server %d has failed\n", servNum)
				update = true
			}
		}

		// Update the entry
		if update == true {
			ml.Servers[servNum].Id = newId
			ml.Servers[servNum].Changed = true
		}

		ml.Servers[servNum].Mutex.Unlock()

		ml.UpdateFTMutex.Lock()
		ml.UpdateFT = true
		ml.UpdateFTMutex.Unlock()
	}
}

// Updates the finger table from the membership list
func (ft *FingerTable) Update() {
	// Set the bool to unchanged
	membershipList.UpdateFTMutex.Lock()
	membershipList.UpdateFT = false
	membershipList.UpdateFTMutex.Unlock()

	self := shared.GetOwnServerNumber()
	powerOfTwo := 1

	addedServers := map[int]bool{}

	for i := 0; i < shared.FingerTableSize; i++ {
		// Find starting point of the search
		start := (self-1+powerOfTwo)%shared.NumServers + 1
		look := start

		// Set finger table entry to invalid
		ft.Entries[i].Number = 0
		ft.Entries[i].Hostname = ""

		// Find first server after start that hasn't already been added
		// 'cont' ensures that the loop is executed at least once
		for cont := true; cont || look != start; cont = false {
			// Check if server is online
			membershipList.Servers[look].Mutex.Lock()
			isOnline := !membershipList.Servers[look].Id.Failed && !membershipList.Servers[look].Id.TimeStamp.IsZero()
			membershipList.Servers[look].Mutex.Unlock()

			// Add server if not yourself, online, and not in another finger table entry
			if self != look && isOnline && !addedServers[look] {
				ft.Entries[i].Number = look
				ft.Entries[i].Hostname = shared.GetServerAddress(look)

				addedServers[look] = true
				break
			}

			// Increment server number and take modulo
			look = (look % shared.NumServers) + 1
		}

		powerOfTwo *= 2
	}
}

func IamIntroducer() bool {
	return shared.GetOwnServerNumber() == Introducer.Number
}

func Join() {
	// Add yourself to your membership table
	servNum := shared.GetOwnServerNumber()
	membershipList.Servers[servNum].Mutex.Lock()
	membershipList.Servers[servNum].Id.ServNum = uint8(servNum)
	membershipList.Servers[servNum].Id.TimeStamp = time.Now()
	membershipList.Servers[servNum].Changed = true
	membershipList.Servers[servNum].Mutex.Unlock()

	membershipList.UpdateFTMutex.Lock()
	membershipList.UpdateFT = true
	membershipList.UpdateFTMutex.Unlock()

	fileLog.Printf("Server %d joining the network\n", servNum)

	if !IamIntroducer() {
		// Contact the introducer
		conn, pingErr := net.Dial("udp", fmt.Sprintf("%s:%d", Introducer.Hostname, shared.SwimIntroducerPort))
		if pingErr != nil {
			log.Fatal("Introducer ping error: ", pingErr)
		}
		defer conn.Close()
		// Just ping the introducer, it will send the mem list back on the same connection
		conn.Write([]byte(""))
	} else {
		// Start ping interval for introducer
		go PingIntervalFunction()
	}
}

// Leave the group by
func LeaveGroup() {
	// Set your own membership entry to failed
	servNum := shared.GetOwnServerNumber()

	membershipList.Servers[servNum].Mutex.Lock()
	membershipList.Servers[servNum].MarkFailed()
	membershipList.Servers[servNum].Mutex.Unlock()

	membershipList.UpdateFTMutex.Lock()
	membershipList.UpdateFT = true
	membershipList.UpdateFTMutex.Unlock()

	// Ping your children
	SendPingAndMembershipList()

	// Stop your own process
	fileLog.Printf("Leaving the failure detector peacefully. Goodbye!")
	log.Println("Ending process normally")
	os.Exit(0)
}

func PingIntervalFunction() {
	for {
		SendPingAndMembershipList()
		time.Sleep(shared.PingInterval)
	}
}

// Send the membership list to children
func SendPingAndMembershipList() {
	// Update finger table only if membership list changed
	// TODO don't call finger tables every time
	//membershipList.UpdateFTMutex.Lock()
	//hasChanged := membershipList.UpdateFT
	//membershipList.UpdateFTMutex.Unlock()
	//if hasChanged == true {
	//fingerTable.Update()
	//}
	fingerTable.Update()

	changedML := membershipList.GetChanged(false)
	//println("Mem list: " + membershipList.Str(true))
	if len(changedML) != 0 {
		fmt.Printf("ChangedML is %v\n", changedML)
		for _, element := range changedML {
			if element.Failed == true {
				NodeFailed = int(element.ServNum)
				NeedAction = true
				fmt.Print("FFFFUCK")
				log.Print("!!!!!!!!!!!!!!!!!!!!!!!sssss")
				NewLeader := electLeader(NodeFailed)
				if NewLeader != Leader && NewLeader == GetHostindex()+1 {
					readFileMapping(FileMap, NodeMap)
				}
				Leader = NewLeader
				log.Printf("the failed node is %d", NodeFailed)
				failRescue(NodeFailed)
				SetNeedAction(false)
			}
		}
	}

	for i, server := range fingerTable.Entries {
		if server.Number == 0 {
			break
		}

		// Unmark this server as received an ACK from it
		membershipList.Servers[server.Number].Mutex.Lock()
		membershipList.Servers[server.Number].ReceivedACK = false
		membershipList.Servers[server.Number].Mutex.Unlock()

		conn, pingErr := net.Dial("udp", fmt.Sprintf("%s:%d", server.Hostname, shared.SwimPingPort))
		if pingErr != nil {
			log.Fatal("SendPingAndMembershipList ping error: ", pingErr)
		}
		defer conn.Close()

		changedMLJSON, _ := json.Marshal(changedML)

		conn.Write(changedMLJSON)
		// If the ACK doesn't come back in time, then mark as failed
		go func(ii int) {
			time.Sleep(shared.ACKTimeout)
			membershipList.Servers[fingerTable.Entries[ii].Number].Mutex.Lock()
			if !membershipList.Servers[fingerTable.Entries[ii].Number].ReceivedACK {
				membershipList.Servers[fingerTable.Entries[ii].Number].MarkFailed()
				fileLog.Printf("I detected server %d has failed\n", fingerTable.Entries[ii].Number)
			}
			membershipList.Servers[fingerTable.Entries[ii].Number].Mutex.Unlock()
		}(i)
	}
}

// Opens the port so that this machine can be pinged by others
func OpenPortForPing() {
	addr := net.UDPAddr{
		Port: shared.SwimPingPort,
	}
	conn, udpErr := net.ListenUDP("udp", &addr)
	if udpErr != nil {
		log.Fatal("OpenPortForPing udp error: ", udpErr)
	}

	go func() {
		for {
			var buf [memListBufferSize]byte
			udpLen, senderAddr, readUDPErr := conn.ReadFromUDP(buf[:])
			if readUDPErr != nil {
				log.Fatal("OpenPortForPing read udp error: ", readUDPErr)
			}
			// fmt.Printf("Received ping from %v\n", senderAddr)

			go func() {
				// Decode a struct sent over the network
				var memIds []MembershipId
				// println("Got a message on the ping port")
				_ = json.Unmarshal(buf[:udpLen], &memIds)
				if len(memIds) != 0 {
					fmt.Printf("Message is %+v\n", memIds)
				}
				membershipList.Update(memIds)
			}()
			go SendACK(conn, senderAddr)
		}
	}()
}

func SendACK(conn *net.UDPConn, addr *net.UDPAddr) {
	addr.Port = shared.SwimACKPort
	_, err := conn.WriteToUDP([]byte(strconv.Itoa(shared.GetOwnServerNumber())), addr)
	if err != nil {
		fmt.Printf("Couldn't send ACK %v", err)
	}
}

// Opens the port so that this machine can receive acknowlegements from pings
func OpenPortForACK() {
	addr := net.UDPAddr{
		Port: shared.SwimACKPort,
	}
	conn, udpErr := net.ListenUDP("udp", &addr)
	if udpErr != nil {
		log.Fatal("OpenPortForACK udp error: ", udpErr)
	}

	go func() {
		for {
			var buf [16]byte
			ackLen, _, readUDPErr := conn.ReadFromUDP(buf[:])
			if readUDPErr != nil {
				log.Fatal("OpenPortForACK read udp error: ", readUDPErr)
			}

			go func() {
				servNum, _ := strconv.Atoi(string(buf[:ackLen]))
				//fmt.Printf("Received ACK from server %d\n", servNum)

				membershipList.Servers[servNum].Mutex.Lock()
				membershipList.Servers[servNum].ReceivedACK = true
				membershipList.Servers[servNum].Mutex.Unlock()
			}()
		}
	}()
}

func OpenPortForIntroducer() {
	addr := net.UDPAddr{
		Port: shared.SwimIntroducerPort,
	}
	introducerCon, udpErr := net.ListenUDP("udp", &addr)
	if udpErr != nil {
		log.Fatal("OpenPortForIntroducer udp error: ", udpErr)
	}

	println("Opening port for introducer")
	go func() {
		for {
			var buf [memListBufferSize]byte
			if !IamIntroducer() {
				introducerCon.SetReadDeadline(time.Now().Add(shared.IntroducerTimeout))
			}
			udpLen, senderAddr, readUDPErr := introducerCon.ReadFromUDP(buf[:])
			if readUDPErr != nil {
				log.Print("OpenPortForIntroducer read udp error: ", readUDPErr)
			}

			if IamIntroducer() {
				println("I'm the introducer! I'll send my membership list to the pingy boi")
				go func() {
					fullMemList := membershipList.GetChanged(true)
					jsonList, _ := json.Marshal(fullMemList)
					introducerCon.Write(jsonList)
					senderAddr.Port = shared.SwimIntroducerPort
					_, _ = introducerCon.WriteToUDP(jsonList, senderAddr)
				}()
			} else {
				println("I'm the pingy boi, and I got the membership list from the introducer")
				fmt.Printf("Length of this message was %d bytes\n", udpLen)
				go func() {
					var memIds []MembershipId
					_ = json.Unmarshal(buf[:udpLen], &memIds)
					membershipList.Update(memIds)
					go PingIntervalFunction()
				}()
				// Close the connection, because the requester will never need to get the mem list again
				introducerCon.Close()
				return
			}
		}
	}()
}

func Initialize() {
	println("Initializing failure detector")
	OpenPortForPing()
	OpenPortForACK()
	OpenPortForIntroducer()
	println("Finished initializing failure detector")

	println("Joining group")
	Join()
	println("Successfully joined group")
}
func PrintMembership() {
	println("Mem list: " + membershipList.Str(true))
}

func GetOnlineMember() []int {
	return membershipList.GetMemberList()
}

func (ml *MembershipList) GetMemberList() (NodeList []int) {
	//var NodeList []int
	for _, entry := range ml.Servers {
		if entry.Id.ServNum != 0 {
			if entry.Id.Failed {

			} else {
				NodeList = append(NodeList, int(entry.Id.ServNum))
			}
		}
	}
	return
}
func GetNeedAction() bool {
	return NeedAction
}
func SetNeedAction(setter bool) {
	NeedAction = setter
	return
}
