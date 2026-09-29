// Package threat 是怪物仇恨表：威胁累积、嘲讽、死亡清理与按时间衰减。
//
// 口径（以 README 为准）：
//   - 坦克造成的威胁 ×2，治疗 ×0.5，其余 ×1。
//   - 嘲讽立刻置顶：被嘲讽者威胁置为当前最高 +1，直到嘲讽结束或它死亡。
//   - 死亡单位从仇恨表清掉，目标永远是存活单位。
//   - 每秒按当前威胁的 5% 衰减，不会减成负数。
//
// 实现：map 存条目、最大堆选目标。加威胁/死亡 O(log n)，选目标 O(1)，
// 内存 O(参与者)。衰减对全体乘同一个正因子，堆序天然保持，无需重建。
package threat

import (
	"container/heap"
	"math"
)

// Entry 是一条仇恨记录。
type Entry struct {
	Name   string
	Threat float64
	Tank   bool
	Healer bool
	Alive  bool
	index  int // 在堆中的下标，-1 表示不在堆里
}

// entryHeap 是按威胁从大到小的最大堆，威胁相同按名字字典序保证确定性。
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
	last := len(old) - 1
	entry := old[last]
	old[last] = nil
	entry.index = -1
	*h = old[:last]
	return entry
}

// Table 是仇恨表。零值即可用。
type Table struct {
	entries   map[string]*Entry
	byThreat  entryHeap
	taunted   string
	tauntLeft float64 // 嘲讽剩余秒数
}

// coefficient 返回职业系数：坦克 ×2、治疗 ×0.5、其余 ×1。
func coefficient(tank, healer bool) float64 {
	switch {
	case tank:
		return 2
	case healer:
		return 0.5
	default:
		return 1
	}
}

// Add 记录一次威胁，按职业系数折算后累加。
func (t *Table) Add(name string, amount float64, tank, healer bool) {
	amount *= coefficient(tank, healer)
	if entry, ok := t.entries[name]; ok {
		entry.Threat += amount
		entry.Tank = entry.Tank || tank
		entry.Healer = entry.Healer || healer
		heap.Fix(&t.byThreat, entry.index)
		return
	}
	if t.entries == nil {
		t.entries = make(map[string]*Entry)
	}
	entry := &Entry{Name: name, Threat: amount, Tank: tank, Healer: healer, Alive: true}
	t.entries[name] = entry
	heap.Push(&t.byThreat, entry)
}

// Kill 标记死亡：把单位从仇恨表里清掉；若它正被嘲讽，嘲讽同时结束。
func (t *Table) Kill(name string) {
	entry, ok := t.entries[name]
	if !ok {
		return
	}
	entry.Alive = false
	heap.Remove(&t.byThreat, entry.index)
	delete(t.entries, name)
	if t.taunted == name {
		t.taunted = ""
		t.tauntLeft = 0
	}
}

// Taunt 嘲讽 name 持续 duration 秒：威胁立即置为当前最高 +1 并置顶，
// 直到嘲讽结束或它死亡。
func (t *Table) Taunt(name string, duration float64) {
	entry, ok := t.entries[name]
	if !ok {
		if t.entries == nil {
			t.entries = make(map[string]*Entry)
		}
		entry = &Entry{Name: name, Alive: true}
		t.entries[name] = entry
		heap.Push(&t.byThreat, entry)
	}
	top := 0.0
	if len(t.byThreat) > 0 {
		top = t.byThreat[0].Threat
	}
	entry.Threat = top + 1
	heap.Fix(&t.byThreat, entry.index)
	t.taunted = name
	t.tauntLeft = duration
}

// Target 返回当前目标：嘲讽未结束且被嘲讽者存活时恒为被嘲讽者，
// 否则为存活单位中威胁最高者。空表返回 ""。
func (t *Table) Target() string {
	if t.taunted != "" && t.tauntLeft > 0 {
		if _, ok := t.entries[t.taunted]; ok {
			return t.taunted
		}
	}
	if len(t.byThreat) == 0 {
		return ""
	}
	return t.byThreat[0].Name
}

// Decay 按时间衰减：每秒衰减当前威胁的 5%（乘 0.95^seconds），不会减成负数；
// 同时推进嘲讽计时。全体乘同一正因子不改变堆序，故无需调整堆。
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
	if t.tauntLeft > 0 {
		t.tauntLeft -= seconds
		if t.tauntLeft <= 0 {
			t.taunted = ""
			t.tauntLeft = 0
		}
	}
}
