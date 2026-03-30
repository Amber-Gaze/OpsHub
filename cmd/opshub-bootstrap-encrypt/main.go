// opshub-bootstrap-encrypt 根据主密钥生成可写入 ops_hub.yaml 的 bootstrap_admin_password_cipher（base64）。
// 示例：
//
//	go run ./cmd/opshub-bootstrap-encrypt -key '你的主密钥' -password '管理员明文密码'
//
// 将输出的整行填入 auth.bootstrap_admin_password_cipher；运行时主密钥需与 -key 一致（环境变量 OPSHUB_BOOTSTRAP_CIPHER_KEY 或配置 bootstrap_cipher_key）。
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Amber-Gaze/OpsHub/internal/pkg/bootstrapcipher"
)

func main() {
	key := flag.String("key", "", "主密钥（与 OPSHUB_BOOTSTRAP_CIPHER_KEY / bootstrap_cipher_key 一致）")
	password := flag.String("password", "", "要加密的明文管理员密码")
	flag.Parse()
	if *key == "" || *password == "" {
		fmt.Fprintln(os.Stderr, "用法: opshub-bootstrap-encrypt -key '主密钥' -password '明文密码'")
		os.Exit(2)
	}
	out, err := bootstrapcipher.Encrypt(*key, *password)
	if err != nil {
		fmt.Fprintln(os.Stderr, "加密失败:", err)
		os.Exit(1)
	}
	fmt.Println(out)
}
