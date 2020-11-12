package measure

import (
	"fmt"
	"strings"
	"time"
)

const (
	summaryPrefix = "|--"
)

// Timer is created for analysis of the function performance
type Timer struct {
	name  string
	start *time.Time
	total time.Duration

	subtimerMap map[string]*Timer
}

// NewTimer create a new timer, only use for root timer only
func NewTimer(name string) *Timer {
	now := time.Now()
	return &Timer{name: name, start: &now, subtimerMap: map[string]*Timer{}}
}

// StartTimer start track time for sub func
func (t *Timer) StartTimer(stacks []string) error {
	stacks, err := t.locateStack(stacks)
	if err != nil {
		return nil
	}

	return t.startSubtimer(stacks)
}

// String implement the print interface
func (t *Timer) String() string {
	return t.summary(0, t.total)
}

// EndTimer always run on end of the func
func (t *Timer) EndTimer(stacks []string) error {
	stacks, err := t.locateStack(stacks)
	if err != nil {
		return err
	}

	return t.endSubtimer(stacks)
}

// IsEnd check wheather the ticker is close
func (t *Timer) IsEnd() bool {
	return t.start == nil
}

func (t *Timer) startSubtimer(stacks []string) error {
	if len(stacks) == 0 {
		if !t.IsEnd() {
			return fmt.Errorf("try to start an unclose timer")
		}

		now := time.Now()
		t.start = &now
		return nil
	}

	nextStack := stacks[len(stacks)-1]
	if _, ok := t.subtimerMap[nextStack]; !ok {
		t.subtimerMap[nextStack] = &Timer{name: nextStack, subtimerMap: map[string]*Timer{}}
	}

	return t.subtimerMap[nextStack].startSubtimer(stacks[:len(stacks)-1])
}

func (t *Timer) endSubtimer(stacks []string) error {
	if len(stacks) == 0 {
		if t.IsEnd() {
			return fmt.Errorf("timer didn't start")
		}

		t.total += time.Now().Sub(*t.start)
		t.start = nil
		return nil
	}

	subtimer, ok := t.subtimerMap[stacks[len(stacks)-1]]
	if !ok {
		return fmt.Errorf("endSubtimer didn't find sub timer")
	}

	return subtimer.endSubtimer(stacks[:len(stacks)-1])
}

// locateStack is using to exclude dust stacks
func (t *Timer) locateStack(stacks []string) ([]string, error) {
	for i := len(stacks) - 1; i >= 0; i-- {
		if stacks[i] == t.name {
			return stacks[:i], nil
		}
	}

	return nil, fmt.Errorf("locateStack didn't match the expect stack")
}

// summary will convert the time spend graph to tree string
func (t *Timer) summary(depth int, parentDuration time.Duration) string {
	result := strings.Repeat("  ", depth) + summaryPrefix
	result += t.name + ": "
	if !t.IsEnd() {
		return result + "<timer didn't ends>\n"
	}

	result += fmt.Sprintf("%s (%.2f)\n", t.total.String(), float64(t.total.Nanoseconds())/float64(parentDuration.Nanoseconds())*100)

	// handle the case that skip middle level time measure case
	nextDepth, total := depth+1, t.total
	if t.total == 0 {
		result = ""
		nextDepth, total = depth, parentDuration
	}

	for _, sub := range t.subtimerMap {
		result += sub.summary(nextDepth, total)
	}

	return result
}
