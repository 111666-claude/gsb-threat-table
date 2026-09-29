package threat

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"
)

func threatOf(t *testing.T, table *Table, name string) float64 {
	t.Helper()
	entry, ok := table.entries[name]
	if !ok {
		t.Fatalf(" entries 里应该有 %s", name)
	}
	return entry.Threat
}

func TestAddAccumulates(t *testing.T) {
	table := &Table{}
	table.Add("a", 30, false, false)
	table.Add("a", 20, false, false)
	if got := threatOf(t, table, "a"); got != 50 {
		t.Fatalf("同一玩家的威胁应该累加：%v", got)
	}
}

func TestCoefficients(t *testing.T) {
	table := &Table{}
	table.Add("tank", 30, true, false)
	table.Add("healer", 100, false, true)
	table.Add("dps", 40, false, false)
	if got := threatOf(t, table, "tank"); got != 60 {
		t.Fatalf("坦克威胁应该 ×2：%v", got)
	}
	if got := threatOf(t, table, "healer"); got != 50 {
		t.Fatalf("治疗威胁应该 ×0.5：%v", got)
	}
	if got := threatOf(t, table, "dps"); got != 40 {
		t.Fatalf("其余职业威胁应该 ×1：%v", got)
	}
}

func TestHighestThreatWins(t *testing.T) {
	table := &Table{}
	table.Add("a", 10, false, false)
	table.Add("b", 40, false, false)
	if table.Target() != "b" {
		t.Fatalf("威胁高的应该是目标：%s", table.Target())
	}
}

func TestEmptyTable(t *testing.T) {
	if (&Table{}).Target() != "" {
		t.Fatal("空表不该有目标")
	}
}

func TestTauntPinsTarget(t *testing.T) {
	table := &Table{}
	table.Add("tank", 30, true, false)    // 60
	table.Add("healer", 100, false, true) // 50
	table.Add("dps", 500, false, false)   // 全场最高
	table.Taunt("tank", 3)
	if table.Target() != "tank" {
		t.Fatalf("被嘲讽者应该立刻成为目标：%s", table.Target())
	}
	if got := threatOf(t, table, "tank"); got != 501 {
		t.Fatalf("嘲讽应把威胁置为当前最高 +1：%v", got)
	}
	// 嘲讽期间别人再加威胁，目标也恒定。
	table.Add("dps", 10000, false, false)
	if table.Target() != "tank" {
		t.Fatalf("嘲讽期间目标应该恒定：%s", table.Target())
	}
	// 嘲讽结束后回到威胁最高者。
	table.Decay(4)
	if table.Target() != "dps" {
		t.Fatalf("嘲讽结束后应该选威胁最高者：%s", table.Target())
	}
}

func TestTauntTargetDies(t *testing.T) {
	table := &Table{}
	table.Add("tank", 30, true, false)
	table.Add("dps", 100, false, false)
	table.Taunt("tank", 10)
	table.Kill("tank")
	if table.Target() != "dps" {
		t.Fatalf("被嘲讽者死亡后应该选存活的最高威胁：%s", table.Target())
	}
}

func TestKillRemovesFromTable(t *testing.T) {
	table := &Table{}
	table.Add("dead-add", 500, false, false)
	table.Add("alive-add", 50, false, false)
	table.Kill("dead-add")
	if table.Target() != "alive-add" {
		t.Fatalf("死人不该是目标：%s", table.Target())
	}
	if _, ok := table.entries["dead-add"]; ok {
		t.Fatal("死亡单位应该从仇恨表里清掉")
	}
	table.Kill("alive-add")
	if table.Target() != "" {
		t.Fatalf("全灭后不该有目标：%s", table.Target())
	}
	// 同名单位重新进表是一条新记录。
	table.Add("alive-add", 5, false, false)
	if table.Target() != "alive-add" {
		t.Fatalf("重新进表的单位应该能当选：%s", table.Target())
	}
}

func TestDecayProportional(t *testing.T) {
	table := &Table{}
	table.Add("a", 100, false, false)
	table.Add("b", 200, false, false)
	table.Decay(1)
	if got := threatOf(t, table, "a"); got != 95 {
		t.Fatalf("每秒应该衰减 5%%：%v", got)
	}
	if got := threatOf(t, table, "b"); got != 190 {
		t.Fatalf("衰减应该按比例而不是固定值：%v", got)
	}
}

func TestDecayNeverNegative(t *testing.T) {
	table := &Table{}
	table.Add("a", 3, false, false)
	table.Decay(1000)
	if got := threatOf(t, table, "a"); got < 0 {
		t.Fatalf("衰减不该为负：%v", got)
	}
}

func TestDeterministic(t *testing.T) {
	run := func() []string {
		table := &Table{}
		var targets []string
		for i := 0; i < 100; i++ {
			table.Add(fmt.Sprintf("p%d", i%7), float64(i%13), i%3 == 0, i%5 == 0)
			if i%11 == 0 {
				table.Taunt(fmt.Sprintf("p%d", i%7), 2)
			}
			if i%17 == 0 {
				table.Kill(fmt.Sprintf("p%d", (i+1)%7))
			}
			table.Decay(0.3)
			targets = append(targets, table.Target())
		}
		return targets
	}
	first, second := run(), run()
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("同一批事件重复执行结果应该相同：第 %d 步 %s != %s", i, first[i], second[i])
		}
	}
}

func TestScale(t *testing.T) {
	table := &Table{}
	start := time.Now()
	for i := 0; i < 100000; i++ {
		table.Add(fmt.Sprintf("p%d", i%1000), 1, i%3 == 0, i%5 == 0)
	}
	_ = table.Target()
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("1000 参与者 10 万次加威胁应在 1 秒内完成：%v", elapsed)
	}
	if len(table.entries) != 1000 {
		t.Fatalf("内存应该 O(参与者)：%d", len(table.entries))
	}
}

// reference 是口径的独立参考实现：按系数累加 + 线性扫描选最大。
type reference struct {
	threat    map[string]float64
	taunted   string
	tauntLeft float64
}

func newReference() *reference {
	return &reference{threat: make(map[string]float64)}
}

func (r *reference) add(name string, amount float64, tank, healer bool) {
	r.threat[name] += amount * coefficient(tank, healer)
}

func (r *reference) kill(name string) {
	delete(r.threat, name)
	if r.taunted == name {
		r.taunted = ""
		r.tauntLeft = 0
	}
}

func (r *reference) taunt(name string, duration float64) {
	top := 0.0
	for _, threat := range r.threat {
		if threat > top {
			top = threat
		}
	}
	r.threat[name] = top + 1
	r.taunted = name
	r.tauntLeft = duration
}

func (r *reference) decay(seconds float64) {
	if seconds <= 0 {
		return
	}
	factor := math.Pow(0.95, seconds)
	for name := range r.threat {
		r.threat[name] *= factor
		if r.threat[name] < 0 {
			r.threat[name] = 0
		}
	}
	if r.tauntLeft > 0 {
		r.tauntLeft -= seconds
		if r.tauntLeft <= 0 {
			r.taunted = ""
			r.tauntLeft = 0
		}
	}
}

func (r *reference) target() string {
	if r.taunted != "" && r.tauntLeft > 0 {
		if _, ok := r.threat[r.taunted]; ok {
			return r.taunted
		}
	}
	best := ""
	bestThreat := math.Inf(-1)
	for name, threat := range r.threat {
		if threat > bestThreat || (threat == bestThreat && name < best) {
			best, bestThreat = name, threat
		}
	}
	return best
}

// TestAgainstReference 随机 200 组「普通攻击 + 治疗 + 嘲讽 + 死亡 + 衰减」
// 事件流，逐步与参考实现比对目标，差异必须为 0。
func TestAgainstReference(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	const participants = 20
	for stream := 0; stream < 200; stream++ {
		table := &Table{}
		ref := newReference()
		name := func() string { return fmt.Sprintf("p%d", rng.Intn(participants)) }
		for step := 0; step < 200; step++ {
			switch rng.Intn(6) {
			case 0, 1: // 普通攻击 / 坦克攻击
				n, amount, tank := name(), rng.Float64()*100, rng.Intn(2) == 0
				table.Add(n, amount, tank, false)
				ref.add(n, amount, tank, false)
			case 2: // 治疗
				n, amount := name(), rng.Float64()*100
				table.Add(n, amount, false, true)
				ref.add(n, amount, false, true)
			case 3: // 嘲讽
				n, duration := name(), rng.Float64()*3
				table.Taunt(n, duration)
				ref.taunt(n, duration)
			case 4: // 死亡
				n := name()
				table.Kill(n)
				ref.kill(n)
			case 5: // 衰减
				seconds := rng.Float64() * 2
				table.Decay(seconds)
				ref.decay(seconds)
			}
			if got, want := table.Target(), ref.target(); got != want {
				t.Fatalf("第 %d 组第 %d 步：got %s, want %s", stream, step, got, want)
			}
		}
	}
}
