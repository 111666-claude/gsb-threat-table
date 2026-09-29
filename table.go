// Package threat 是怪物仇恨表：记录每个玩家造成的威胁并挑选当前目标。
// 口径：坦克威胁 ×2、治疗 ×0.5；嘲讽置顶（威胁置为当前最高 +1）；
// 死亡单位立即清出仇恨表；每秒按当前威胁的 5% 比例衰减且不为负。
package threat

import (
	"container/heap"
	"math"
)

// Entry 是一条仇恨记录。死亡单位不会留在表里。
type Entry struct {
	Name   string
	Threat float64
	Tank   bool
	Healer bool
	index  int // 在堆中的下标，-1 表示不在堆里
}

// entryHeap 是按威胁值排序的最大堆，威胁相同按名字字典序，保证结果确定。
type entryHeap []*Entry

func (h entryHeap) Len() int { return len(h) }

func (h entryHeap) Less(i, j int) bool {
	if h[i].Threat != h[j].Threat {
		return h[i].Threat > h[j].Threat
	}
	return h[i].Name < h[j].Name
}

func (h entryHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *entryHeap) Push(x any) {
	entry := x.(*Entry)
	entry.index = len(*h)
	*h = append(*h, entry)
}

func (h *entryHeap) Pop() any {
	old := *h
	n := len(old)
	entry := old[n-1]
	old[n-1] = nil
	entry.index = -1
	*h = old[:n-1]
	return entry
}

// Table 是仇恨表。内存占用 O(参与者)，选目标 O(1)，加威胁 O(log n)。
type Table struct {
	entries map[string]*Entry
	top     entryHeap
	taunted string
}

// New 建一张空仇恨表。
func New() *Table {
	return &Table{entries: make(map[string]*Entry)}
}

// Add 记录一次威胁：坦克 ×2、治疗 ×0.5、其余 ×1，同一玩家只增不减。
func (t *Table) Add(name string, amount float64, tank, healer bool) {
	if tank {
		amount *= 2
	}
	if healer {
		amount *= 0.5
	}
	t.lazyInit()
	if entry, ok := t.entries[name]; ok {
		entry.Threat += amount
		entry.Tank = entry.Tank || tank
		entry.Healer = entry.Healer || healer
		heap.Fix(&t.top, entry.index)
		return
	}
	entry := &Entry{Name: name, Threat: amount, Tank: tank, Healer: healer}
	t.entries[name] = entry
	heap.Push(&t.top, entry)
}

// Kill 把死亡单位从仇恨表里清掉；若它正被嘲讽，嘲讽同时结束。
func (t *Table) Kill(name string) {
	entry, ok := t.entries[name]
	if !ok {
		return
	}
	heap.Remove(&t.top, entry.index)
	delete(t.entries, name)
	if t.taunted == name {
		t.taunted = ""
	}
}

// Taunt 嘲讽：目标立刻置顶，威胁值置为当前最高 +1，直到嘲讽结束或它死亡。
func (t *Table) Taunt(name string) {
	entry, ok := t.entries[name]
	if !ok {
		return
	}
	entry.Threat = t.maxThreat() + 1
	heap.Fix(&t.top, entry.index)
	t.taunted = name
}

// EndTaunt 结束当前嘲讽。
func (t *Table) EndTaunt() {
	t.taunted = ""
}

// Target 选目标：嘲讽期间恒为被嘲讽者，否则是威胁最高的存活单位。
func (t *Table) Target() string {
	if t.taunted != "" {
		if _, ok := t.entries[t.taunted]; ok {
			return t.taunted
		}
		t.taunted = ""
	}
	if len(t.top) == 0 {
		return ""
	}
	return t.top[0].Name
}

// Decay 按时间衰减：每秒衰减当前威胁的 5%，按比例复合，不会减成负数。
func (t *Table) Decay(seconds float64) {
	if seconds <= 0 {
		return
	}
	factor := math.Pow(0.95, seconds)
	for _, entry := range t.entries {
		entry.Threat *= factor
		if entry.Threat < 0 {
			entry.Threat = 0
		}
	}
	heap.Init(&t.top)
}

// Threat 返回某玩家当前的威胁值，供测试与调试。
func (t *Table) Threat(name string) float64 {
	if entry, ok := t.entries[name]; ok {
		return entry.Threat
	}
	return 0
}

// Len 返回表里存活单位的数量。
func (t *Table) Len() int {
	return len(t.entries)
}

func (t *Table) maxThreat() float64 {
	max := 0.0
	for _, entry := range t.entries {
		if entry.Threat > max {
			max = entry.Threat
		}
	}
	return max
}

func (t *Table) lazyInit() {
	if t.entries == nil {
		t.entries = make(map[string]*Entry)
	}
}
