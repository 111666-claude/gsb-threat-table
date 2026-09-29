package threat

import "testing"

func TestAddAccumulates(t *testing.T) {
	table := &Table{}
	table.Add("tank", 30, true, false)
	table.Add("tank", 20, true, false)
	if len(table.entries) != 1 || table.entries[0].Threat != 50 {
		t.Fatalf("同一玩家的威胁应该累加：%+v", table.entries)
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

func TestDecayFloors(t *testing.T) {
	table := &Table{}
	table.Add("a", 3, false, false)
	table.Decay(10)
	if table.entries[0].Threat != 0 {
		t.Fatalf("衰减不该为负：%v", table.entries[0].Threat)
	}
}
