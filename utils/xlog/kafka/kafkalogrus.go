package kafkalogrus

import (
	"errors"
	"os"
	"xway/utils/mq"

	"github.com/sirupsen/logrus"
)

type KafkaLogrusHook struct {
	id             string
	defaultTopic   string
	injectHostname bool
	hostname       string
	levels         []logrus.Level
	formatter      logrus.Formatter
	producer       *mq.MqProducer
}

func NewKafkaLogrusHook(id string, levels []logrus.Level, formatter logrus.Formatter, producer *mq.MqProducer, defaultTopic string, injectHostname bool) (*KafkaLogrusHook, error) {
	var err error
	var hostname string
	if hostname, err = os.Hostname(); err != nil {
		hostname = "localhost"
	}

	hook := &KafkaLogrusHook{
		id,
		defaultTopic,
		injectHostname,
		hostname,
		levels,
		formatter,
		producer,
	}
	return hook, nil
}

func (hook *KafkaLogrusHook) Id() string {
	return hook.id
}

func (hook *KafkaLogrusHook) Levels() []logrus.Level {
	return hook.levels
}

func (hook *KafkaLogrusHook) Fire(entry *logrus.Entry) error {
	var key string
	var b []byte
	var err error

	// t, _:=entry.Data["time"].(time.Time)
	// if b,err=t.MarshalBinary();err!=nil{
	// 	return err
	// }

	topic := hook.defaultTopic
	if tsRaw, ok := entry.Data["topic"]; ok {
		if ts, ok := tsRaw.(string); !ok {
			return errors.New("Incorrect topic filed type (should be string)")
		} else {
			topic = ts
		}
	}

	if k, ok := entry.Data["key"]; ok {
		if v, ok := k.(string); !ok {
			return errors.New("Incorrect key filed type (should be string)")
		} else {
			key = v
		}
	}

	if hook.injectHostname {
		if _, ok := entry.Data["host"]; !ok {
			entry.Data["host"] = hook.hostname
		}
	}

	if b, err = hook.formatter.Format(entry); err != nil {
		return err
	}

	// value:=sarama.ByteEncoder(b)
	value := string(b)

	hook.producer.SendMessageAsync(&mq.Message{
		Topic:   topic,
		Key:     key,
		Content: value,
	})
	return nil
}
