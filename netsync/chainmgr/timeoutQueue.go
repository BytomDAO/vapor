package chainmgr

import (
	"sync"
	"time"
)

type timeoutQueue struct {
	times    []*time.Time
	timeMap  map[string]*time.Time
	duration time.Duration
	mu       sync.Mutex
}

func newTimeoutQueue(duration time.Duration) *timeoutQueue {
	return &timeoutQueue{
		times:    []*time.Time{},
		timeMap:  make(map[string]*time.Time),
		duration: duration,
	}
}

func (tq *timeoutQueue) addTimer(id string) {
	tq.mu.Lock()
	tq.mu.Unlock()

	time := time.Now().Add(tq.duration)
	tq.timeMap[id] = &time
	tq.times = append(tq.times, &time)
}

func (tq *timeoutQueue) delTimer(id string) {
	tq.mu.Lock()
	tq.mu.Unlock()

	target, ok := tq.timeMap[id]
	if !ok {
		return
	}

	for i, time := range tq.times {
		if time == target {
			tq.times = append(tq.times[:i], tq.times[i+1:]...)
			delete(tq.timeMap, id)
			break
		}
	}

	return
}

func (tq *timeoutQueue) getNextTimeoutDuration() *time.Duration {
	tq.mu.Lock()
	tq.mu.Unlock()

	var duration *time.Duration
	if len(tq.times) > 0 {
		d := tq.times[0].Sub(time.Now())
		duration = &d
	}

	return duration
}

func (tq *timeoutQueue) getFirstTimeoutID() *string {
	tq.mu.Lock()
	tq.mu.Unlock()

	if len(tq.times) == 0 {
		return nil
	}

	timeoutTime := tq.times[0]
	for id, time := range tq.timeMap {
		if time == timeoutTime {
			return &id
		}
	}

	return nil
}