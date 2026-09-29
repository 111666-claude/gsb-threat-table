// 命令行入口：演示嘲讽置顶与死亡清理两个场景。
package main

import (
	"flag"
	"fmt"
	"os"

	"example.com/threat"
)

func main() {
	sample := flag.String("sample", "", "演示场景：taunt 或 dead")
	flag.Parse()

	switch *sample {
	case "taunt":
		// 坦克 30 点（×2）、治疗 100 点（×0.5），随后嘲讽坦克。
		table := &threat.Table{}
		table.Add("tank", 30, true, false)
		table.Add("healer", 100, false, true)
		table.Taunt("tank", 3)
		fmt.Printf("target=%s\n", table.Target())
	case "dead":
		// 500 威胁的假人已死、50 威胁的活人还在。
		table := &threat.Table{}
		table.Add("dead-add", 500, false, false)
		table.Add("alive-add", 50, false, false)
		table.Kill("dead-add")
		fmt.Printf("target=%s\n", table.Target())
	default:
		fmt.Fprintln(os.Stderr, "usage: threat-table --sample taunt|dead")
		os.Exit(2)
	}
}
