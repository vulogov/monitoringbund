package bund

import (
	"github.com/nats-io/nats.go"
	"github.com/pieterclaerhout/go-log"
)


func NRBundAgent(m *nats.Msg) {
	if ! HadSync {
		log.Warn("Request received but agent not in SYNC state. Request ignored.")
		return
	}
	msg := UnMarshal(m.Data)
	if msg == nil {
		log.Error("Invalid packet received")
	}
	if msg.PktKey == "Agitator" && len(msg.Value) > 0 {
		log.Debugf("Script: %v", msg.Uri)
		BundGlobalEvalExpression(string(msg.Value), msg.Args, msg.Res)
	}
}

func Agent() {
	Init()
	InitEtcdAgent("agent")
	UpdateLocalConfigFromEtcd()
	InitNatsAgent()
	if ! WaitSync() {
		return
	}
	InitStoragePipe()
	log.Debugf("[ MBUND ] bund.Agent(%v) is reached", ApplicationId)
	NatsRecv(NRBundAgent)
	Loop()
}
