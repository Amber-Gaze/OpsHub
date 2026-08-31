// config-consumer 演示「专门的下游服务」从配置中心拉取配置：
//   - 登录 IAM 获取令牌
//   - 通过 configclient 封装（authorize 取 scope + 携带 X-Auth-* 头）调用 /configs/pull
//   - 支持按层级（business/module/name、path、key）精确拉取
//   - 基于全局版本号做「增量判断更新」：首次全量，之后 PullSince 只拉变更项与被删 key
//
// 用法（先启动 IAM + 配置中心，并配置好 bootstrap 管理员或普通用户）：
//
//	go run ./examples/config-consumer -user admin -pass '你的密码' \
//	    -auth http://127.0.0.1:8101 -config http://127.0.0.1:8201 -interval 5s
//	go run ./examples/config-consumer ... -path pay/gateway   # 只关注某层级
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/Amber-Gaze/OpsHub/pkg/configclient"
)

func main() {
	authURL := flag.String("auth", "http://127.0.0.1:8101", "IAM 服务根地址")
	configURL := flag.String("config", "http://127.0.0.1:8201", "配置中心根地址（或网关 http://127.0.0.1:8001）")
	user := flag.String("user", "", "登录用户名（与 -pass 一起用；或改用下方服务凭证）")
	pass := flag.String("pass", "", "登录密码")
	accessKeyID := flag.String("access-key-id", "", "服务凭证 AccessKeyID（替代账号密码）")
	accessKeySecret := flag.String("access-key-secret", "", "服务凭证 AccessKeySecret")
	interval := flag.Duration("interval", 5*time.Second, "拉取间隔")
	path := flag.String("path", "", "可选：只关注某层级前缀（如 pay/gateway）")
	modules := flag.String("modules", "", "订阅模块列表（逗号分隔，如 pay/gateway,common/ratelimit）")
	defaultDataDir := ""
	if exe, err := os.Executable(); err == nil {
		// 程序在 ./bin/ 目录运行，落盘数据放同级 ./data/
		defaultDataDir = filepath.Join(filepath.Dir(exe), "..", "data")
	}
	dataDir := flag.String("data-dir", defaultDataDir, "落盘数据目录（默认与 bin 平级的 data/）")
	flag.Parse()

	// 解析订阅模块列表
	var mods []string
	for _, m := range strings.Split(*modules, ",") {
		m = strings.Trim(strings.TrimSpace(m), "/")
		if m != "" {
			mods = append(mods, m)
		}
	}
	if len(mods) > 0 {
		fmt.Printf("== 订阅模块: %s；落盘目录: %s\n", strings.Join(mods, ", "), *dataDir)
	}

	if (*user == "" || *pass == "") && (*accessKeyID == "" || *accessKeySecret == "") {
		fmt.Fprintln(os.Stderr, "需要 -user/-pass（账号登录）或 -access-key-id/-access-key-secret（服务凭证登录）")
		flag.Usage()
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cli := configclient.New(*authURL, *configURL)
	if *accessKeyID != "" {
		// 服务凭证登录：无需明文密码，凭证可独立轮换/吊销
		if err := cli.LoginWithAccessKey(ctx, *accessKeyID, *accessKeySecret); err != nil {
			fmt.Fprintln(os.Stderr, "access key login failed:", err)
			os.Exit(1)
		}
		fmt.Printf("== 服务凭证登录成功，从 %s 拉取配置，间隔 %s\n", *configURL, *interval)
	} else {
		if err := cli.Login(ctx, *user, *pass); err != nil {
			fmt.Fprintln(os.Stderr, "login failed:", err)
			os.Exit(1)
		}
		fmt.Printf("== 账号登录成功，从 %s 拉取配置，间隔 %s\n", *configURL, *interval)
	}

	// 演示：按层级精确拉取（一次性展示）
	if *path != "" {
		if items, err := cli.PullPath(ctx, *path); err != nil {
			fmt.Printf("[warn] PullPath(%q): %v\n", *path, err)
		} else {
			fmt.Printf("== 按层级 %q 拉取（%d 项）==\n", *path, len(items))
			printSorted(toMap(items))
		}
	}

	// 增量拉取：rev 记录已应用的全局版本号；0 表示尚未拉取（首次全量）
	local := map[string]configclient.Item{}
	rev := int64(0)

	for {
		res, err := cli.PullSince(ctx, rev)
		if err != nil {
			fmt.Println("[error] pull:", err)
		} else if !res.HasChanged(rev) {
			fmt.Printf("== 无变化（revision=%d）==\n", res.Revision)
		} else if rev == 0 {
			local = toMap(res.Items)
			local = filterToModules(local, mods)
			rev = res.Revision
			fmt.Printf("== 首次快照（revision=%d，%d 项）==\n", rev, len(local))
			printSorted(local)
			writeLocal(*dataDir, local)
		} else {
			inc := applyIncremental(local, res)
			fmt.Printf("== 增量（revision %d -> %d）==\n", rev, res.Revision)
			if len(inc.added)+len(inc.updated)+len(inc.removed) == 0 {
				fmt.Println("  无实际变化（版本号前进但键值未变）")
			} else {
				for _, k := range inc.added {
					fmt.Printf("  [新增] %s = %q (v%d)\n", k, local[k].Value, local[k].Version)
				}
				for _, k := range inc.updated {
					fmt.Printf("  [更新] %s = %q (v%d)\n", k, local[k].Value, local[k].Version)
				}
				for _, k := range inc.removed {
					fmt.Printf("  [删除] %s\n", k)
				}
			}
			rev = res.Revision
			local = filterToModules(local, mods)
			writeLocal(*dataDir, local)
		}

		select {
		case <-ctx.Done():
			fmt.Println("== 退出 ==")
			return
		case <-time.After(*interval):
		}
	}
}

// incResult 一次增量应用后的变化清单（用于打印）。
type incResult struct {
	added, updated, removed []string
}

// filterToModules 只保留订阅模块范围内的配置项；mods 为空表示不限制。
func filterToModules(local map[string]configclient.Item, mods []string) map[string]configclient.Item {
	if len(mods) == 0 {
		return local
	}
	out := map[string]configclient.Item{}
	for k, v := range local {
		for _, m := range mods {
			if k == m || strings.HasPrefix(k, m+"/") {
				out[k] = v
				break
			}
		}
	}
	return out
}

// writeLocal 把本地快照落盘为 JSON（WriteTo 按业务/模块分组写文件）。
func writeLocal(dir string, local map[string]configclient.Item) {
	if dir == "" {
		return
	}
	items := make([]configclient.Item, 0, len(local))
	for _, it := range local {
		items = append(items, it)
	}
	if err := configclient.WriteTo(items, dir); err != nil {
		fmt.Println("[warn] write data:", err)
	} else {
		fmt.Printf("== 已落盘 %d 项配置到 %s ==\n", len(items), dir)
	}
}

// applyIncremental 把增量结果应用到本地快照，返回本次新增/更新/删除的 key。
func applyIncremental(local map[string]configclient.Item, res *configclient.PullResult) incResult {
	var r incResult
	for _, it := range res.Items {
		old, ok := local[it.Key]
		local[it.Key] = it
		switch {
		case !ok:
			r.added = append(r.added, it.Key)
		case old.Version != it.Version:
			r.updated = append(r.updated, it.Key)
		}
	}
	for _, k := range res.Removed {
		if _, ok := local[k]; ok {
			r.removed = append(r.removed, k)
		}
		delete(local, k)
	}
	sort.Strings(r.added)
	sort.Strings(r.updated)
	sort.Strings(r.removed)
	return r
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
