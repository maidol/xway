package mq

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"

	"github.com/Shopify/sarama"
)

type MqProducer struct {
	syncProducer  sarama.SyncProducer
	asyncProducer sarama.AsyncProducer
}

type Message struct {
	Topic   string
	Key     string
	Content string
}

func NewProducer(cfg *MqConfig) *MqProducer {
	fmt.Println("[mq config] ", cfg)
	p := newSyncProducer(cfg)
	ap := newAsyncProducer(cfg)
	return &MqProducer{
		syncProducer:  p,
		asyncProducer: ap,
	}
}

func newConfig(cfg *MqConfig) *sarama.Config {
	mqConfig := sarama.NewConfig()
	mqConfig.ChannelBufferSize = 1024
	mqConfig.Net.SASL.Enable = true
	mqConfig.Net.SASL.User = cfg.Ak
	mqConfig.Net.SASL.Password = cfg.Password
	mqConfig.Net.SASL.Handshake = true

	certBytes, err := ioutil.ReadFile(GetFullPath(cfg.CertFile))
	if err != nil {
		panic("kafka producer ioutil.ReadFile failed to read " + GetFullPath(cfg.CertFile))
	}
	clientCertPool := x509.NewCertPool()
	ok := clientCertPool.AppendCertsFromPEM(certBytes)
	if !ok {
		panic("kafka producer failed to parse root certificate")
	}

	mqConfig.Net.TLS.Config = &tls.Config{
		// Certificates: []tls.Certificate{},
		RootCAs:            clientCertPool,
		InsecureSkipVerify: true,
	}

	mqConfig.Net.TLS.Enable = true
	mqConfig.Producer.Return.Successes = true
	mqConfig.Producer.Return.Errors = true

	if err = mqConfig.Validate(); err != nil {
		msg := fmt.Sprintf("Kafka producer config invalidate. config: %v. err: %v", *cfg, err)
		fmt.Println(msg)
		panic(msg)
	}

	return mqConfig
}

func newSyncProducer(cfg *MqConfig) sarama.SyncProducer {
	var producer sarama.SyncProducer
	mqConfig := newConfig(cfg)
	producer, err := sarama.NewSyncProducer(cfg.Servers, mqConfig)
	if err != nil {
		msg := fmt.Sprintf("Kafka producer create fail. err: %v", err)
		fmt.Println(msg)
		panic(msg)
	}

	return producer
}

func newAsyncProducer(cfg *MqConfig) sarama.AsyncProducer {
	var producer sarama.AsyncProducer
	mqConfig := newConfig(cfg)
	producer, err := sarama.NewAsyncProducer(cfg.Servers, mqConfig)
	if err != nil {
		msg := fmt.Sprintf("Kafka async producer create fail. err: %v", err)
		fmt.Println(msg)
		panic(msg)
	}

	// TODO: 发生错误时, 要考虑安全退出
	go func() {
		for {
			select {
			case <-producer.Successes():
			// case smsg := <-producer.Successes():
			// 	msg := fmt.Sprintf("Kafka async producer send msg success. topic: %v. key: %v. content: %v.", smsg.Topic, smsg.Key, smsg.Value)
			// 	fmt.Println(msg)
			case emsg := <-producer.Errors():
				msg := fmt.Sprintf("Kafka async producer send message error. err: %v. topic: %v. key: %v. content: %v", emsg.Error(), emsg.Msg.Topic, emsg.Msg.Key, emsg.Msg.Value)
				fmt.Println(msg)
			}
		}
	}()

	return producer
}

func (mqp *MqProducer) SendMessage(m *Message) error {
	msg := &sarama.ProducerMessage{
		Topic: m.Topic,
		Key:   sarama.StringEncoder(m.Key),
		Value: sarama.StringEncoder(m.Content),
	}

	_, _, err := mqp.syncProducer.SendMessage(msg)
	if err != nil {
		msg := fmt.Sprintf("Kafka producer send message error. err: %v. topic: %v. key: %v. content: %v", err, m.Topic, m.Key, m.Content)
		fmt.Println(msg)
		return err
	}

	return nil
}

func (mqp *MqProducer) SendMessageAsync(m *Message) {
	msg := &sarama.ProducerMessage{
		Topic: m.Topic,
		Key:   sarama.StringEncoder(m.Key),
		Value: sarama.StringEncoder(m.Content),
	}
	mqp.asyncProducer.Input() <- msg
}
