package remotelist

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
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

	l.LogOperation(fmt.Sprintf("append %d %d\n", list_id, value))

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

	l.LogOperation(fmt.Sprintf("remove %d\n", list_id))

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
	f, _ := os.OpenFile("/app/data/log.txt", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)

	rl := &RemoteList{
		lists:   make(map[int]*InstancedList),
		logFile: f,
	}
	rl.LoadSnapshot()
	rl.RebuildFromLog()
	go rl.RunSnapshotRoutine()
	return rl
}
func (l *RemoteList) RebuildFromLog() {
	metaData, err := os.ReadFile("/app/data/snapshot_meta.txt")
	var snapshotTime int64 = 0
	if err == nil {
		snapshotTime, _ = strconv.ParseInt(string(metaData), 10, 64)
	}

	scanner := bufio.NewScanner(l.logFile)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)

		if len(parts) < 2 {
			continue
		}

		logTime, _ := strconv.ParseInt(parts[0], 10, 64)
		if logTime <= snapshotTime {
			continue
		}

		op := parts[1]
		listID, _ := strconv.Atoi(parts[2])

		if _, exists := l.lists[listID]; !exists {
			l.lists[listID] = &InstancedList{}
			l.size++
		}

		switch op {
		case "append":
			val, _ := strconv.Atoi(parts[3])
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

func (l *RemoteList) LogOperation(op string) {
	timestamp := time.Now().UnixNano()
	logLine := fmt.Sprintf("%d %s", timestamp, op)
	l.logFile.WriteString(logLine + "\n")
}

func (l *RemoteList) SaveSnapshot() error {
	l.mu.Lock()

	listsCopy := make(map[int][]int)
	for id, inst := range l.lists {
		inst.mu.Lock()
		listCopy := make([]int, len(inst.list))
		copy(listCopy, inst.list)
		listsCopy[id] = listCopy
		inst.mu.Unlock()
	}
	l.mu.Unlock()

	f, err := os.Create("/app/data/snapshot.json")
	if err != nil {
		return err
	}
	defer f.Close()

	metaFile, err := os.Create("/app/data/napshot_meta.txt")
	if err != nil {
		return err
	}
	defer metaFile.Close()

	_, err = fmt.Fprintf(metaFile, "%d", time.Now().UnixNano())
	encoder := json.NewEncoder(f)
	return encoder.Encode(listsCopy)
}

func (l *RemoteList) LoadSnapshot() {
	f, err := os.Open("/app/data/snapshot.json")
	if err != nil {
		return
	}
	defer f.Close()

	var snapshot map[int][]int
	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&snapshot); err != nil {
		return
	}

	for id, values := range snapshot {
		inst := &InstancedList{
			list: values,
			size: uint32(len(values)),
		}
		l.lists[id] = inst
		l.size++
	}
}

func (l *RemoteList) RunSnapshotRoutine() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		err := l.SaveSnapshot()
		if err != nil {
			fmt.Println("Erro ao salvar snapshot:", err)
		} else {
			fmt.Println("Snapshot salvo com sucesso.")
		}
	}
}

func (l *RemoteList) Close() {
	if l.logFile != nil {
		l.logFile.Close()
	}
}
