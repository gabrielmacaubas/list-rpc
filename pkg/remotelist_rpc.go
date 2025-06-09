package remotelist

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

type RemoteList struct {
	mu      sync.Mutex
	lists   map[int]*InstancedList
	size    uint32
	logFile *os.File
}

type InstancedList struct {
	mu   sync.Mutex
	list []int
	size uint32
}

func (l *RemoteList) Append(args [2]int, reply *bool) error {
	list_id := args[0]
	value := args[1]
	l.mu.Lock()
	instancedList := l.lists[list_id]
	defer l.mu.Unlock()

	instancedList.mu.Lock()
	instancedList.list = append(instancedList.list, value)
	instancedList.size++
	instancedList.mu.Unlock()

	l.logOperation(fmt.Sprintf("append %d %d\n", list_id, value))

	*reply = true
	return nil
}

func (l *RemoteList) Get(args [2]int, reply *int) error {
	list_id := args[0]
	i := args[1]
	l.mu.Lock()
	instancedList := l.lists[list_id]
	defer l.mu.Unlock()

	instancedList.mu.Lock()
	defer instancedList.mu.Unlock()

	*reply = instancedList.list[i]
	return nil
}

func (l *RemoteList) Remove(list_id int, reply *int) error {
	l.mu.Lock()
	instancedList := l.lists[list_id]
	defer l.mu.Unlock()

	instancedList.mu.Lock()
	defer instancedList.mu.Unlock()

	n := len(instancedList.list)
	*reply = instancedList.list[n-1]
	instancedList.list = instancedList.list[:n-1]
	instancedList.size--

	l.logOperation(fmt.Sprintf("remove %d\n", list_id))

	return nil
}

func (l *RemoteList) Size(list_id int, reply *uint32) error {
	l.mu.Lock()
	instancedList := l.lists[list_id]
	l.mu.Unlock()

	instancedList.mu.Lock()
	defer instancedList.mu.Unlock()

	*reply = instancedList.size
	return nil
}

func (l *RemoteList) AddList(_ struct{}, reply *int) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	newID := len(l.lists)
	if l.lists == nil {
		l.lists = make(map[int]*InstancedList)
	}

	l.lists[newID] = &InstancedList{
		list: []int{},
		size: 0,
	}
	l.size++

	*reply = newID
	return nil
}

func NewRemoteList() *RemoteList {
	f, _ := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)

	rl := &RemoteList{
		lists:   make(map[int]*InstancedList),
		logFile: f,
	}
	rl.rebuildFromLog()

	return rl
}

func (l *RemoteList) rebuildFromLog() {
	scanner := bufio.NewScanner(l.logFile)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)

		op := parts[0]
		listID, _ := strconv.Atoi(parts[1])

		if _, exists := l.lists[listID]; !exists {
			l.lists[listID] = &InstancedList{}
			l.size++
		}

		switch op {
		case "append":
			val, _ := strconv.Atoi(parts[2])
			l.lists[listID].list = append(l.lists[listID].list, val)
			l.lists[listID].size++
		case "remove":
			inst := l.lists[listID]
			n := len(inst.list)
			if n > 0 {
				inst.list = inst.list[:n-1]
				inst.size--
			}
		}
	}

	fmt.Println("Estado reconstruído das listas:")
	for id, list := range l.lists {
		fmt.Printf("Lista %d: %v (tamanho: %d)\n", id, list.list, list.size)
	}
}

func (l *RemoteList) logOperation(op string) {
	l.logFile.WriteString(op)
}

func (l *RemoteList) Close() {
	if l.logFile != nil {
		l.logFile.Close()
	}
}
