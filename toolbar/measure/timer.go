package measure

import (
	"fmt"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	summaryPrefix = "|--"
)

// Timer is created for record the func performance status as the tree struct
type Timer struct {
	name        string
	start       time.Time
	end         time.Time
	subtimerMap map[string]*Timer
}

// NewTimer create a new timer
func NewTimer(name string) *Timer {
	return &Timer{name: name, start: time.Now(), subtimerMap: map[string]*Timer{}}
}

// AddSubtimer start track time for sub func
func (t *Timer) AddSubtimer(stacks []string) error {
	stacks, err := t.locateStack(stacks)
	if err != nil {
		return nil
	}

	t.addSubtimer(stacks)
	return nil
}

// EndTimer always run on end of the func
func (t *Timer) EndTimer(stacks []string) error {
	stacks, err := t.locateStack(stacks)
	if err != nil {
		return err
	}

	if err := t.endTimer(stacks); err != nil {
		return err
	}

	if len(stacks) == 0 {
		log.WithField("module", logModule).Info(t.summary(0, nil))
	}
	return nil
}

func (t *Timer) addSubtimer(stacks []string) {
	if len(stacks) == 0 {
		return
	}

	nextStack := stacks[len(stacks)-1]
	if _, ok := t.subtimerMap[nextStack]; !ok {
		t.subtimerMap[nextStack] = NewTimer(nextStack)
	}

	t.subtimerMap[nextStack].addSubtimer(stacks[:len(stacks)-1])
}

func (t *Timer) endTimer(stacks []string) error {
	if len(stacks) == 0 {
		t.end = time.Now()
		return nil
	}

	nextStack := stacks[len(stacks)-1]
	subtimer, ok := t.subtimerMap[nextStack]
	if !ok {
		return fmt.Errorf("endTimer didn't find sub timer")
	}

	return subtimer.endTimer(stacks[:len(stacks)-1])
}

// locateStack is use for elitimate dust stacks
func (t *Timer) locateStack(stacks []string) ([]string, error) {
	for i := len(stacks) - 1; i >= 0; i-- {
		if stacks[i] == t.name {
			return stacks[:i], nil
		}
	}

	return nil, fmt.Errorf("locateStack didn't match the expect stack")
}

// summary will convert the time spend graph to tree string
func (t *Timer) summary(depth int, parentDuration *time.Duration) string {
	result := strings.Repeat("	", depth) + summaryPrefix
	result += t.name + ": "
	if t.start.IsZero() || t.end.IsZero() {
		return result + "<timer didn't ends>\n"
	}

	duration := t.end.Sub(t.start)
	if parentDuration == nil {
		parentDuration = &duration
	}

	result += fmt.Sprintf("%s (%.2f)\n", duration.String(), float64(duration.Nanoseconds())/float64(parentDuration.Nanoseconds())*100)
	for _, sub := range t.subtimerMap {
		result += sub.summary(depth+1, &duration)
	}

	return result
}
