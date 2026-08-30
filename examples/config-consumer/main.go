// config-consumer 演示「专门的下游服务」从配置中心拉取配置：
//   - 登录 IAM 获取令牌
//   - 通过 configclient 封装（authorize 取 scope + 携带 X-Auth-* 头）调用 /configs/pull
//   - 周期性拉取，并对比两次快照输出 新增/更新/删除（按 version 判定）
//
// 用法（先启动 IAM + 配置中心，并配置好 bootstrap 管理员或普通用户）：
//
//	go run ./examples/config-consumer -user admin -pass '你的密码' \
//	    -auth http://127.0.0.1:8004 -config http://127.0.0.1:8007 -interval 5s
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	"github.com/Amber-Gaze/OpsHub/pkg/configclient"
)

func main() {
	authURL := flag.String("auth", "http://127.0.0.1:8004", "IAM 服务根地址")
	configURL := flag.String("config", "http://127.0.0.1:8007", "配置中心根地址（或网关 http://127.0.0.1:8001）")
	user := flag.String("user", "", "登录用户名")
	pass := flag.String("pass", "", "登录密码")
	interval := flag.Duration("interval", 5*time.Second, "拉取间隔")
	flag.Parse()

	if *user == "" || *pass == "" {
		fmt.Fprintln(os.Stderr, "需要 -user 与 -pass")
		flag.Usage()
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cli := configclient.New(*authURL, *configURL)
	if err := cli.Login(ctx, *user, *pass); err != nil {
		fmt.Fprintln(os.Stderr, "login failed:", err)
		os.Exit(1)
	}
	fmt.Printf("== 登录成功，开始从 %s 拉取配置，间隔 %s\n", *configURL, *interval)

	var prev map[string]configclient.Item
	for {
		items, err := cli.Pull(ctx)
		if err != nil {
			fmt.Println("[error] pull:", err)
		} else {
			cur := toMap(items)
			switch {
			case prev == nil:
				fmt.Println("== 首次快照 ==")
				printSorted(cur)
			default:
				added, updated, removed := configclient.Diff(prev, cur)
				fmt.Printf("== 变更检测 ==")
				if len(added)+len(updated)+len(removed) == 0 {
					fmt.Println(" 无变化")
				} else {
					fmt.Println()
					for _, k := range added {
						fmt.Printf("  [新增] %s = %q (v%d)\n", k, cur[k].Value, cur[k].Version)
					}
					for _, k := range updated {
						fmt.Printf("  [更新] %s : v%d -> v%d = %q\n", k, prev[k].Version, cur[k].Version, cur[k].Value)
					}
					for _, k := range removed {
						fmt.Printf("  [删除] %s (was v%d)\n", k, prev[k].Version)
					}
				}
			}
			prev = cur
		}

		select {
		case <-ctx.Done():
			fmt.Println("== 退出 ==")
			return
		case <-time.After(*interval):
		}
	}
}

func toMap(items []configclient.Item) map[string]configclient.Item {
	out := make(map[string]configclient.Item, len(items))
	for _, it := range items {
		out[it.Key] = it
	}
	return out
}

func printSorted(m map[string]configclient.Item) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %-40s = %-16q (v%d, by %s)\n", k, m[k].Value, m[k].Version, m[k].UpdatedBy)
	}
}
