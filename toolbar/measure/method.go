package measure

import (
	"fmt"
	"runtime/debug"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
)

const (
	logModule = "measure"
)

var store sync.Map

// Start trigger record of stack trace run time record as graph view
func Start() {
	routineID, stacks, err := stacks()
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on measure get stacks")
		return
	}

	data, ok := store.Load(routineID)
	if !ok {
		store.Store(routineID, NewTimer(stacks[0]))
		return
	}

	if err := data.(*Timer).AddSubtimer(stacks); err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err, "routine": routineID, "stack": stacks}).Error("fail on add sub timer")
	}
}

// End end the stack trace run time
func End() {
	routineID, stacks, err := stacks()
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on measure get stacks")
		return
	}

	data, ok := store.Load(routineID)
	if !ok {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("fail on measure timer by routine ID")
		return
	}

	if err := data.(*Timer).EndTimer(stacks); err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err, "routine": routineID, "stack": stacks}).Error("fail on end timer")
	}
}

func stacks() (string, []string, error) {
	stacks := []string{}
	for _, stack := range strings.Split(string(debug.Stack()), "\n") {
		// skip the file path stack
		if strings.HasPrefix(stack, "	") {
			continue
		}

		// delete the func memory address stuff
		if subPos := strings.LastIndexAny(stack, "("); subPos > 0 {
			stacks = append(stacks, stack[:subPos])
		} else {
			stacks = append(stacks, stack)
		}
	}

	if len(stacks) < 4 {
		return "", nil, fmt.Errorf("fail to decode stack")
	}

	return stacks[0], stacks[4:], nil
}
