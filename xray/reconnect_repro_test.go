package xray

import "testing"

// 复现「同进程 Stop→Run 第二次起核心」是否失败。
// 用带真实监听端口的 socks inbound（空 inbound 证明不了端口/listener 残留）。
const reproConfig = `{
  "log": {"loglevel": "warning"},
  "inbounds": [{
    "tag": "socks-in",
    "protocol": "socks",
    "listen": "127.0.0.1",
    "port": 47821,
    "settings": {"auth": "noauth", "udp": true}
  }],
  "outbounds": [{"protocol": "freedom", "tag": "direct"}]
}`

func TestReproSameProcessRestart(t *testing.T) {
	_ = StopXray()

	// 第 1 轮
	if err := RunXrayFromJSON(reproConfig); err != nil {
		t.Fatalf("round1 start: %v", err)
	}
	if !GetXrayState() {
		t.Fatal("round1 should be running")
	}
	if err := StopXray(); err != nil {
		t.Fatalf("round1 stop: %v", err)
	}
	if GetXrayState() {
		t.Fatal("round1 should be stopped after StopXray")
	}
	t.Log("round1 ok (started, stopped)")

	// 第 2 轮：同进程立刻再起，模拟「停→连」
	if err := RunXrayFromJSON(reproConfig); err != nil {
		t.Fatalf("round2 start FAILED (this is the bug): %v", err)
	}
	t.Log("round2 started OK — same-process restart works")
	_ = StopXray()
}
