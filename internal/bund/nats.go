package bund

import (
	"os"
	"fmt"
	"time"
	"github.com/nats-io/nats.go"
	"github.com/vulogov/monitoringbund/internal/conf"
	"github.com/vulogov/monitoringbund/internal/signal"
	"github.com/pieterclaerhout/go-log"
	"github.com/bamzi/jobrunner"
)

var Nats *nats.Conn
var QueueName string
var SysQueueName string
var EvtQueueName string
var MetricQueueName string
var LogQueueName string
var TraceQueueName string
var HadSync bool
var NatsCluster string

func SysQueueHandler(m *nats.Msg) {
	msg := UnMarshal(m.Data)
	if msg == nil {
		log.Error("Invalid packet received")
	}
	if IfSTOP(msg) {
		log.Infof("STOP(%v) message received. Wait for Cluster to exit", msg.PktId)
		time.Sleep(*conf.Timeout)
		return
	}
	if IfSYNC(msg) {
		if ! HadSync {
			if msg.ApplicationId() == ApplicationId {
				log.Debugf("SYNC(%v) message from itself is not triggered SYNC state for %v", msg.PktId, ApplicationId)
			} else {
				HadSync = true
				log.Infof("SYNC(%v) message triggered SYNC state for %v", msg.ApplicationId(), ApplicationId)
			}
		}
		return
	}
	if IfMSG(msg) {
		log.Info(string(msg.Value))
		return
	}
}

func InitNatsAgent() {
	var err error

	log.Debug("Configuring NATS cluster config")
	NatsCluster = fmt.Sprintf("%v ", *conf.Gnats)
	for _, c := range *conf.GnatsC {
		NatsCluster += fmt.Sprintf(", %v", c)
	}
	log.Debugf("Connecting to NATS: %v", NatsCluster)
	Nats, err = nats.Connect(
		NatsCluster,
		nats.Name(ApplicationId),
		nats.ReconnectWait(*conf.Timeout),
		nats.PingInterval(*conf.Timeout),
		nats.Timeout(*conf.Timeout),
	)
	if err != nil {
		log.Errorf("[ NATS ] %v", err)
		signal.ExitRequest()
		os.Exit(10)
	}
	QueueName 		= fmt.Sprintf("%s:%s", *conf.Id, *conf.Name)
	SysQueueName 	= fmt.Sprintf("%s:%s:sys", *conf.Id, *conf.Name)
	EvtQueueName 	= fmt.Sprintf("%s:%s:evt", *conf.Id, *conf.Name)
	MetricQueueName 	= fmt.Sprintf("%s:%s:metric", *conf.Id, *conf.Name)
	LogQueueName 	= fmt.Sprintf("%s:%s:log", *conf.Id, *conf.Name)
	TraceQueueName 	= fmt.Sprintf("%s:%s:trace", *conf.Id, *conf.Name)
	log.Debugf("Queue name: %v", QueueName)
	log.Debugf("SysQueue name: %v", SysQueueName)
	log.Debugf("Metric Queue name: %v", MetricQueueName)
	log.Debugf("Event Queue name: %v", EvtQueueName)
	log.Debugf("Log Queue name: %v", LogQueueName)
	log.Debugf("Trace Queue name: %v", TraceQueueName)
	NatsRecvSys(SysQueueHandler)
	jobrunner.Schedule("@every 5s", NATSSync{})
}

func NatsSend(data []byte) {
	if DoContinue {
		Nats.Publish(QueueName, data)
	}
}

func NatsSendSys(data []byte) {
	if DoContinue {
		Nats.Publish(SysQueueName, data)
	}
}

func NatsRecv(fun nats.MsgHandler) {
	Nats.QueueSubscribe(QueueName, *conf.Id, fun)
}

func NatsTelemetryRecv(fun nats.MsgHandler) {
	Nats.QueueSubscribe(EvtQueueName, *conf.Id, fun)
	Nats.QueueSubscribe(MetricQueueName, *conf.Id, fun)
	Nats.QueueSubscribe(LogQueueName, *conf.Id, fun)
	Nats.QueueSubscribe(TraceQueueName, *conf.Id, fun)
}

func NatsRecvSys(fun nats.MsgHandler) {
	Nats.Subscribe(SysQueueName, fun)
}

func CloseNatsAgent() {
	log.Debugf("Terminating and draining NATS session")
	Nats.Flush()
}

func init() {
	HadSync = false
}
