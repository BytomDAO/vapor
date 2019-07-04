package chainmgr

import "sync"

const (
	unstart  = 0
	process  = 1
	complete = 2

	maxRequestNum = 3
)

type headersTask struct {
	status        int
	requestNumber uint64
}

func (h *headersTask) addRequestNum() {
	h.requestNumber++
	if h.requestNumber >= maxRequestNum {
		h.status = complete
	}
}

func (h *headersTask) setStatus(status int) {
	h.status = status
}

type headersTasks struct {
	mtx   sync.RWMutex
	tasks map[string]*headersTask
}

func newHeadersTasks() *headersTasks {
	return &headersTasks{
		tasks: make(map[string]*headersTask),
	}
}

func (ht *headersTasks) addTask(id string) {
	ht.mtx.Lock()
	defer ht.mtx.Unlock()

	ht.tasks[id] = &headersTask{}
}

func (ht *headersTasks) addRequestNum(id string) {
	ht.mtx.Lock()
	defer ht.mtx.Unlock()

	task, ok := ht.tasks[id]
	if !ok {
		return
	}
	task.addRequestNum()
}

func (ht *headersTasks) getPeers(status int) []string {
	ht.mtx.RLock()
	defer ht.mtx.RUnlock()

	result := []string{}
	for id, task := range ht.tasks {
		if task.status == status {
			result = append(result, id)
		}
	}
	return result
}

func (ht *headersTasks) isRequestedPeer(id string) bool {
	ht.mtx.RLock()
	defer ht.mtx.RUnlock()

	_, ok := ht.tasks[id]
	return ok
}

func (ht *headersTasks) setStatus(id string, status int) {
	ht.mtx.Lock()
	defer ht.mtx.Unlock()

	task, ok := ht.tasks[id]
	if !ok {
		return
	}
	task.setStatus(status)
}
