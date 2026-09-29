package threat

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"
)

func TestAddAccumulates(t *testing.T) {
	table := New()
	table.Add("a", 30, false, false)
	table.Add("a", 20, false, false)
	if table.Len() != 1 || table.Threat("a") != 50 {
		t.Fatalf("同一玩家的威胁应该累加：%v", table.Threat("a"))
	}
}

func TestClassCoefficients(t *testing.T) {
	table := New()
	table.Add("tank", 30, true, false)
	table.Add("healer", 100, false, true)
	table.Add("dps", 40, false, false)
	if got := table.Threat("tank"); got != 60 {
		t.Fatalf("坦克威胁应该 ×2：%v", got)
	}
	if got := table.Threat("healer"); got != 50 {
		t.Fatalf("治疗威胁应该 ×0.5：%v", got)
	}
	if got := table.Threat("dps"); got != 40 {
		t.Fatalf("其余职业威胁应该 ×1：%v", got)
	}
}

func TestHighestThreatWins(t *testing.T) {
	table := New()
	table.Add("a", 10, false, false)
	table.Add("b", 40, false, false)
	if table.Target() != "b" {
		t.Fatalf("威胁高的应该是目标：%s", table.Target())
	}
}

func TestEmptyTable(t *testing.T) {
	if New().Target() != "" {
		t.Fatal("空表不该有目标")
	}
}

func TestTauntPinsTarget(t *testing.T) {
	table := New()
	table.Add("tank", 30, true, false)    // 60
	table.Add("healer", 100, false, true) // 50
	table.Add("dps", 200, false, false)   // 200，原本最高
	table.Taunt("tank")
	if table.Target() != "tank" {
		t.Fatalf("被嘲讽者应该立刻成为目标：%s", table.Target())
	}
	if got := table.Threat("tank"); got != 201 {
		t.Fatalf("嘲讽后威胁应该置为当前最高 +1：%v", got)
	}
	// 嘲讽期间其他人加威胁，目标恒定。
	table.Add("dps", 500, false, false)
	if table.Target() != "tank" {
		t.Fatalf("嘲讽期间目标应该恒定：%s", table.Target())
	}
	table.EndTaunt()
	if table.Target() != "dps" {
		t.Fatalf("嘲讽结束后应该回到威胁最高者：%s", table.Target())
	}
}

func TestTauntEndsOnDeath(t *testing.T) {
	table := New()
	table.Add("tank", 30, true, false)
	table.Add("dps", 100, false, false)
	table.Taunt("tank")
	table.Kill("tank")
	if table.Target() != "dps" {
		t.Fatalf("被嘲讽者死亡后应该切到下一个存活单位：%s", table.Target())
	}
}

func TestKillRemovesFromTable(t *testing.T) {
	table := New()
	table.Add("dead-add", 500, false, false)
	table.Add("alive-add", 50, false, false)
	table.Kill("dead-add")
	if table.Target() != "alive-add" {
		t.Fatalf("死人不该是目标：%s", table.Target())
	}
	if table.Len() != 1 || table.Threat("dead-add") != 0 {
		t.Fatalf("死人不该留在表里：len=%d", table.Len())
	}
	table.Kill("alive-add")
	if table.Target() != "" {
		t.Fatalf("全灭后不该有目标：%s", table.Target())
	}
}

func TestDecayProportional(t *testing.T) {
	table := New()
	table.Add("a", 200, false, false)
	table.Decay(1)
	if got, want := table.Threat("a"), 200*0.95; got != want {
		t.Fatalf("每秒应该衰减当前值的 5%%：got %v want %v", got, want)
	}
	table.Decay(1)
	if got, want := table.Threat("a"), 200*0.95*0.95; math.Abs(got-want) > 1e-9 {
		t.Fatalf("衰减应该按比例复合：got %v want %v", got, want)
	}
}

func TestDecayFloors(t *testing.T) {
	table := New()
	table.Add("a", 3, false, false)
	table.Decay(10000)
	if table.Threat("a") < 0 {
		t.Fatalf("衰减不该为负：%v", table.Threat("a"))
	}
}

func TestDeterministic(t *testing.T) {
	run := func() string {
		table := New()
		table.Add("a", 10, true, false)
		table.Add("b", 15, false, true)
		table.Add("c", 12, false, false)
		table.Decay(0.5)
		table.Taunt("b")
		return table.Target()
	}
	if run() != run() {
		t.Fatal("同一批事件重复执行结果应该相同")
	}
}

// reference 是「按系数累加 + 线性扫描取最大」的参考实现，用于对照自证。
type reference struct {
	threat  map[string]float64
	taunted string
}

func newReference() *reference {
	return &reference{threat: make(map[string]float64)}
}

func (r *reference) add(name string, amount float64, tank, healer bool) {
	if tank {
		amount *= 2
	}
	if healer {
		amount *= 0.5
	}
	r.threat[name] += amount
}

func (r *reference) kill(name string) {
	delete(r.threat, name)
	if r.taunted == name {
		r.taunted = ""
	}
}

func (r *reference) taunt(name string) {
	if _, ok := r.threat[name]; !ok {
		return
	}
	max := 0.0
	for _, v := range r.threat {
		if v > max {
			max = v
		}
	}
	r.threat[name] = max + 1
	r.taunted = name
}

func (r *reference) decay(seconds float64) {
	factor := math.Pow(0.95, seconds)
	for name := range r.threat {
		r.threat[name] *= factor
	}
}

func (r *reference) target() string {
	if r.taunted != "" {
		if _, ok := r.threat[r.taunted]; ok {
			return r.taunted
		}
		r.taunted = ""
	}
	best, bestThreat := "", -1.0
	// 名字有序遍历，保证平局时与堆实现的字典序口径一致。
	names := make([]string, 0, len(r.threat))
	for name := range r.threat {
		names = append(names, name)
	}
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if names[j] < names[i] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}
	for _, name := range names {
		if r.threat[name] > bestThreat {
			best, bestThreat = name, r.threat[name]
		}
	}
	return best
}

func TestAgainstReference(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	for stream := 0; stream < 200; stream++ {
		table := New()
		ref := newReference()
		var names []string
		events := 50 + rng.Intn(150)
		for step := 0; step < events; step++ {
			if len(names) == 0 || (len(names) < 20 && rng.Intn(3) == 0) {
				names = append(names, fmt.Sprintf("p%d", len(names)))
			}
			pick := func() string { return names[rng.Intn(len(names))] }
			switch rng.Intn(6) {
			case 0, 1: // 普通攻击
				name := pick()
				amount := rng.Float64() * 100
				table.Add(name, amount, false, false)
				ref.add(name, amount, false, false)
			case 2: // 治疗
				name := pick()
				amount := rng.Float64() * 100
				table.Add(name, amount, false, true)
				ref.add(name, amount, false, true)
			case 3: // 嘲讽（含坦克系数的攻击也混在这里）
				name := pick()
				if rng.Intn(2) == 0 {
					amount := rng.Float64() * 50
					table.Add(name, amount, true, false)
					ref.add(name, amount, true, false)
				}
				table.Taunt(name)
				ref.taunt(name)
			case 4: // 死亡
				name := pick()
				table.Kill(name)
				ref.kill(name)
			case 5: // 衰减
				seconds := rng.Float64() * 2
				table.Decay(seconds)
				ref.decay(seconds)
			}
			if got, want := table.Target(), ref.target(); got != want {
				t.Fatalf("stream %d step %d：目标不一致 got=%s want=%s", stream, step, got, want)
			}
		}
	}
}

func TestScale(t *testing.T) {
	table := New()
	start := time.Now()
	for i := 0; i < 100000; i++ {
		name := fmt.Sprintf("p%d", i%1000)
		table.Add(name, 1, i%7 == 0, i%5 == 0)
		if i%997 == 0 && table.Target() == "" {
			t.Fatal("有存活单位时应该有目标")
		}
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("10 万次加威胁应在 1 秒内完成：%v", elapsed)
	}
	if table.Len() != 1000 {
		t.Fatalf("内存应按参与者计：%d", table.Len())
	}
}
