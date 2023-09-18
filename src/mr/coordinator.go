package mr

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"
)

const (
	IDLE = iota
	IN_PROGRESS
	COMPLETED
)

type MapTask struct {
	filename string
	index    int
}

var maptasks chan MapTask
var reducetasks chan int

type Coordinator struct {
	mapTaskStatus     map[string]int
	reduceTaskStatus  map[int]int
	finish            bool
	inputFiles        []string
	nReduce           int
	mapIndex          int
	reduceIndex       int
	intermediateFiles [][]string
	RWMutexLock       *sync.RWMutex
	mapCompleted      bool
	reduceCompleted   bool
}

// Your code here -- RPC handlers for the worker to call.
func (c *Coordinator) DistributeTask(args *MrArgs, reply *MrReply) error {
	select {
	case mapTask := <-maptasks:
		reply.MapFileName = mapTask.filename
		reply.Index = mapTask.index
		reply.TaskType = "map"
		reply.NReduce = c.nReduce
		c.RWMutexLock.Lock()
		c.mapTaskStatus[mapTask.filename] = IN_PROGRESS
		c.RWMutexLock.Unlock()
		go c.watchWorkerMap(mapTask) //goroutine to monitor if the worker finishes its job in 10 secs
		return nil
	case reduceNumber := <-reducetasks:
		reply.Files = c.intermediateFiles[reduceNumber]
		reply.Index = reduceNumber
		reply.TaskType = "reduce"
		c.RWMutexLock.Lock()
		c.reduceTaskStatus[reduceNumber] = IN_PROGRESS
		c.RWMutexLock.Unlock()
		go c.watchWorkerReduce(reduceNumber)
		return nil
	}
	return nil
}

func (m *Coordinator) watchWorkerMap(task MapTask) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	// if worker completes MapTask within 10 seconds, set status to COMPLETED else set it to IDLE
	// if status is IDLE, the MapTask gets assigned to workers again
	for {
		select {
		case <-ticker.C:
			m.RWMutexLock.Lock()
			m.mapTaskStatus[task.filename] = IDLE
			m.RWMutexLock.Unlock()
			maptasks <- task
		default:
			m.RWMutexLock.RLock()
			if m.mapTaskStatus[task.filename] == COMPLETED {
				m.RWMutexLock.RUnlock()
				return
			}
			m.RWMutexLock.RUnlock()
		}
	}
}

func (c *Coordinator) NotifyMapSuccess(args *NotifyMapSuccessArgs, reply *NotifyReply) error {
	c.RWMutexLock.Lock()
	defer c.RWMutexLock.Unlock()
	c.mapTaskStatus[args.File] = COMPLETED
	completed := true
	for _, v := range c.mapTaskStatus {
		if v != COMPLETED {
			completed = false
			break
		}
	}
	c.mapCompleted = completed
	if c.mapCompleted {
		for i := 0; i < c.nReduce; i++ {
			c.reduceTaskStatus[i] = IDLE
			reducetasks <- i
		}
	}
	return nil
}

// if all MapTasks are completed, then ReduceTasks are added to the channel
// the workers will now be assigned ReduceTasks
func (m *Coordinator) watchWorkerReduce(reduceNumber int) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			m.RWMutexLock.Lock()
			m.reduceTaskStatus[reduceNumber] = IDLE
			m.RWMutexLock.Unlock()
			reducetasks <- reduceNumber
		default:
			m.RWMutexLock.RLock()
			if m.reduceTaskStatus[reduceNumber] == COMPLETED {
				m.RWMutexLock.RUnlock()
				return
			}
			m.RWMutexLock.RUnlock()
		}
	}
}

func (c *Coordinator) NotifyReduceSuccess(args *NotifyReduceSuccessArgs, reply *NotifyReply) error {
	c.RWMutexLock.Lock()
	defer c.RWMutexLock.Unlock()
	c.reduceTaskStatus[args.ReduceIndex] = COMPLETED
	completed := true
	for _, v := range c.reduceTaskStatus {
		if v != COMPLETED {
			completed = false
			break
		}
	}
	c.reduceCompleted = completed
	return nil
}

func (c *Coordinator) NotifyIntermediateFile(args *NotifyIntermediateArgs, reply *NotifyReply) error {
	c.RWMutexLock.Lock()
	defer c.RWMutexLock.Unlock()
	c.intermediateFiles[args.ReduceIndex] = append(c.intermediateFiles[args.ReduceIndex], args.File)
	return nil
}

// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	c.RWMutexLock.Lock()
	defer c.RWMutexLock.Unlock()
	ret := c.reduceCompleted // if c.reduceCompleted is true, then the mrcoordinator's process is completed

	return ret
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{}

	maptasks = make(chan MapTask, len(files))
	reducetasks = make(chan int, nReduce)
	c.mapTaskStatus = make(map[string]int, len(files))
	c.reduceTaskStatus = make(map[int]int, nReduce)
	for index, file := range files {
		c.mapTaskStatus[file] = IDLE
		mapTask := MapTask{}
		mapTask.index = index
		mapTask.filename = file
		maptasks <- mapTask
	}

	c.inputFiles = files
	c.nReduce = nReduce
	c.intermediateFiles = make([][]string, nReduce)
	c.RWMutexLock = new(sync.RWMutex)

	c.server()
	return &c
}
