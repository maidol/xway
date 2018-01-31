package redislogrus

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/sirupsen/logrus"
)

type Hook struct {
	id             string
	defaultTopic   string
	injectHostname bool
	hostname       string
	levels         []logrus.Level
	formatter      logrus.Formatter
	producer       *redis.Pool
}

func NewHook(id string, levels []logrus.Level, formatter logrus.Formatter, producer *redis.Pool, defaultTopic string, injectHostname bool) (*Hook, error) {
	var err error
	var hostname string
	if hostname, err = os.Hostname(); err != nil {
		hostname = "localhost"
	}

	hook := &Hook{
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

func (hook *Hook) Id() string {
	return hook.id
}

func (hook *Hook) Levels() []logrus.Level {
	return hook.levels
}

func (hook *Hook) Fire(entry *logrus.Entry) error {
	var key string
	var b []byte
	var err error

	t, _ := entry.Data["time"].(time.Time)
	if b, err = t.MarshalBinary(); err != nil {
		return err
	}

	if hook.injectHostname {
		if _, ok := entry.Data["hostname"]; !ok {
			entry.Data["hostname"] = hook.hostname
		}
	}

	topic := hook.defaultTopic
	if tsRaw, ok := entry.Data["topic"]; ok {
		if ts, ok := tsRaw.(string); !ok {
			return errors.New("Incorrect topic filed type (should be string)")
		} else {
			if ts != "" {
				topic = ts
			}
		}
	}
	entry.Data["topic"] = topic

	key = topic + ":" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if k, ok := entry.Data["key"]; ok {
		if v, ok := k.(string); !ok {
			return errors.New("Incorrect key filed type (should be string)")
		} else {
			if v != "" {
				key = v
			}
		}
	}
	entry.Data["key"] = key

	if b, err = hook.formatter.Format(entry); err != nil {
		return err
	}
	// value:=sarama.ByteEncoder(b)
	value := string(b)

	rdc := hook.producer.Get()
	defer rdc.Close()
	_, re := rdc.Do("SET", key, value)
	if re != nil {
		fmt.Println("[redislogrus.Hook] redis rdc.Do(SET) err: ", re)
	}
	return nil
}
