# threat-table

怪物仇恨表：威胁累积、嘲讽、死亡清理与按时间衰减，只用 Go 标准库。

```
go test ./...
go run ./cmd/threat-table --sample taunt
go run ./cmd/threat-table --sample dead
```

## 口径（README 为准）

- **嘲讽要真的置顶**：被嘲讽者立刻成为目标，威胁值置为当前最高 +1，直到嘲讽结束或它死亡。
  `--sample taunt`（坦克 30 点、治疗 100 点，随后嘲讽坦克）按口径是 `target=tank`，
  现在嘲讽只记了个名字，给 `target=healer`。
- **死亡单位要清出仇恨表**：`--sample dead`（500 威胁的假人已死、50 威胁的活人还在）按口径是
  `target=alive-add`，现在照样选死人，给 `target=dead-add`。
- **职业系数**：坦克造成的威胁 ×2，治疗的威胁 ×0.5，其余 ×1（现在的实现完全没算）。
- **衰减按比例**：每秒按当前威胁的 5% 衰减（现在是每秒固定扣 1 点），且衰减不会让威胁变负。
- 不变量：目标永远是存活单位；同一玩家的威胁只增不减（除衰减）；嘲讽期间目标恒定；
  同一批事件重复执行结果相同。
- 规模：1000 个参与者、每秒 10 万次加威胁 1 秒内完成，内存 O(参与者)；不许每次选目标都全表排序。

## 现在的行为

- `Add` 完全忽略 `tank`/`healer` 系数，所有人的威胁都按原始数字累加。
- `Taunt` 只写 `t.taunted`，`Target` 也不读它。
- `Target` 不检查 `Alive`，死人的威胁还留在表里。
- `Decay` 每秒固定扣 1 点。

## 输出

```
target=tank
target=alive-add
```

## 目录

```
table.go            仇恨表
cmd/threat-table    命令行入口
table_test.go       go test 用例
```
