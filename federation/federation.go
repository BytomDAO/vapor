package federation

import (
	"sort"

	"github.com/vapor/federation/config"
)

func ParseFedProg(warders []config.Warder) []byte {
	SortWarders(warders)

	// TODO:
	return []byte{}
}

type ByPosition []config.Warder

func (w ByPosition) Len() int           { return len(w) }
func (w ByPosition) Swap(i, j int)      { w[i], w[j] = w[j], w[i] }
func (w ByPosition) Less(i, j int) bool { return w[i].Position < w[j].Position }

func SortWarders(warders []config.Warder) []config.Warder {
	sort.Sort(ByPosition(warders))
	return warders
}
