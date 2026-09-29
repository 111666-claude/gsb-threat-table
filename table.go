// Package threat 是怪物仇恨表：记录每个玩家造成的威胁并挑选当前目标。
// 缺陷：忽略坦克/治疗系数、嘲讽只记名字不置顶、目标选择不过滤死亡单位、衰减不按时间比例。
package threat

// Entry 是一条仇恨记录。
type Entry struct {
	Name   string
	Threat float64
	Tank   bool
	Healer bool
	Alive  bool
}

// Table 是仇恨表。
type Table struct {
	entries []Entry
	taunted string
}

// Add 记录一次威胁。缺陷：坦克的 2 倍与治疗的 0.5 倍系数都没算。
func (t *Table) Add(name string, amount float64, tank, healer bool) {
	for index := range t.entries {
		if t.entries[index].Name == name {
			t.entries[index].Threat += amount
			t.entries[index].Tank = t.entries[index].Tank || tank
			t.entries[index].Healer = t.entries[index].Healer || healer
			return
		}
	}
	t.entries = append(t.entries, Entry{Name: name, Threat: amount, Tank: tank, Healer: healer, Alive: true})
}

// Kill 标记死亡。缺陷：只置 Alive=false，不从表里清掉。
func (t *Table) Kill(name string) {
	for index := range t.entries {
		if t.entries[index].Name == name {
			t.entries[index].Alive = false
		}
	}
}

// Taunt 嘲讽。缺陷：只记下名字，威胁值不动，Target 也不看它。
func (t *Table) Taunt(name string) {
	t.taunted = name
}

// Target 选目标。缺陷：直接取威胁最大的那条，不看嘲讽也不看 Alive。
func (t *Table) Target() string {
	best := ""
	bestThreat := -1.0
	for _, entry := range t.entries {
		if entry.Threat > bestThreat {
			best, bestThreat = entry.Name, entry.Threat
		}
	}
	return best
}

// Decay 按时间衰减。缺陷：每秒固定扣 1 点，与「按当前值的 5% 衰减」的口径不符。
func (t *Table) Decay(seconds float64) {
	for index := range t.entries {
		t.entries[index].Threat -= seconds
		if t.entries[index].Threat < 0 {
			t.entries[index].Threat = 0
		}
	}
}
