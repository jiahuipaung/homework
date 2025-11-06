###### 2022-12-19 14:00
***尚未支持剪枝***

```
go test -bench=Benchmark_ExtractWithHandle -run=none -count=3 -benchmem
goos: darwin
goarch: amd64
pkg: github.com/flashcatcloud/fc-insight/logic/logevent
cpu: VirtualApple @ 2.50GHz
Benchmark_ExtractWithHandle-8              10362            115967 ns/op           55433 B/op        439 allocs/op
Benchmark_ExtractWithHandle-8               9624            120296 ns/op           55176 B/op        439 allocs/op
Benchmark_ExtractWithHandle-8               9439            120461 ns/op           55204 B/op        439 allocs/op
```


###### 2022-12-20 11:00
***支持剪枝***

* 大量的剪枝配置时, CPU消耗变化不大, 但内存消耗有增加(存在大量的剪枝行为)

```
go test -run=none -bench=WithHandle -benchmem -count=3
goos: darwin
goarch: amd64
pkg: github.com/flashcatcloud/fc-insight/logic/logevent
cpu: VirtualApple @ 2.50GHz
Benchmark_ExtractWithHandle-8              10323            119865 ns/op           55446 B/op        439 allocs/op
Benchmark_ExtractWithHandle-8               8790            121209 ns/op           55149 B/op        439 allocs/op
Benchmark_ExtractWithHandle-8               8557            122188 ns/op           55155 B/op        439 allocs/op
Benchmark_PruneWithHandle-8                 8956            123309 ns/op           58333 B/op        649 allocs/op
Benchmark_PruneWithHandle-8                 8978            121874 ns/op           58402 B/op        649 allocs/op
Benchmark_PruneWithHandle-8                 9165            121665 ns/op           58366 B/op        649 allocs/op
```


###### 2022-12-21 16:00
***支持剪枝***

* 剪枝的遍历做了一些优化, 对于不需要遍历的地方忽略掉

```
go test -run=none -bench=WithHandle -benchmem -count=3
goos: darwin
goarch: amd64
pkg: github.com/flashcatcloud/fc-insight/logic/logevent
cpu: VirtualApple @ 2.50GHz
Benchmark_ExtractWithHandle-8               9684            115290 ns/op           55522 B/op        439 allocs/op
Benchmark_ExtractWithHandle-8               9889            122664 ns/op           55334 B/op        439 allocs/op
Benchmark_ExtractWithHandle-8               9010            120908 ns/op           55123 B/op        439 allocs/op
Benchmark_PruneWithHandle-8                 9784            117857 ns/op           57449 B/op        578 allocs/op
Benchmark_PruneWithHandle-8                 9520            119635 ns/op           57393 B/op        578 allocs/op
Benchmark_PruneWithHandle-8                 9590            119290 ns/op           57440 B/op        578 allocs/op
```

###### 2022-12-21 16:00
***支持剪枝***

* PruneWithHandle对每一个字段都处理, 处理规则上百条, 原始字段全部重命名
* PruneWithHandle2只对某几个字段做处理, 处理规则7条, 原始字段全部被保留

```
go test -run=none -bench=PruneWithHandle -benchmem -count=3
goos: darwin
goarch: amd64
pkg: github.com/flashcatcloud/fc-insight/logic/logevent
cpu: VirtualApple @ 2.50GHz
Benchmark_PruneWithHandle-8         7652            130864 ns/op           57447 B/op        578 allocs/op
Benchmark_PruneWithHandle-8         9348            141672 ns/op           57490 B/op        578 allocs/op
Benchmark_PruneWithHandle-8         8262            129600 ns/op           57398 B/op        578 allocs/op
Benchmark_PruneWithHandle2-8       15951             76738 ns/op           30580 B/op        344 allocs/op
Benchmark_PruneWithHandle2-8       16479             73445 ns/op           30592 B/op        344 allocs/op
Benchmark_PruneWithHandle2-8       17022             70487 ns/op           30587 B/op        344 allocs/op
```