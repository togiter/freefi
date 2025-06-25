package redis

import (
	"encoding/json"
	"fmt"
	"freefi/trademgr/pkg/logger"
	"time"

	"github.com/gomodule/redigo/redis"
)

// 参考: https://blog.csdn.net/yangxiaodong88/article/details/90730370

type RedisCfg struct {
	IP          string `json:"ip" yaml:"ip"`
	Port        int    `json:"port" yaml:"port"`
	Password    string `json:"password" yaml:"password"`
	DBIndex     int    `json:"dbIndex" yaml:"dbIndex"`
	MaxIdle     int    `json:"maxIdle" yaml:"maxIdle"`
	MaxActive   int    `json:"maxActive" yaml:"maxActive"`
	IdleTimeout int    `json:"idleTimeOut" yaml:"idleTimeOut"`
}

var redisPool *redis.Pool

func Init(cfg RedisCfg) error {
	redisPool = &redis.Pool{
		Dial: func() (conn redis.Conn, e error) {
			optDB := redis.DialDatabase(cfg.DBIndex)
			optPWD := redis.DialPassword(cfg.Password)
			conn, e = redis.Dial("tcp", fmt.Sprintf("%s:%d", cfg.IP, cfg.Port), optDB, optPWD)
			return
		},
		MaxIdle:     cfg.MaxIdle,
		MaxActive:   cfg.MaxActive,
		IdleTimeout: time.Duration(cfg.IdleTimeout) * time.Second,
	}
	return redisPool.Get().Err()
}

func GetGlobalRedis() *redis.Pool {
	return redisPool
}

// Publish 发布消息
func Publish(channel string, message interface{}) error {
	conn := redisPool.Get()
	defer conn.Close()
	msg, err := json.Marshal(message)
	if err != nil {
		return err
	}
	_, err = conn.Do("PUBLISH", channel, msg)
	return err
}

// Subscribe 订阅消息
func Subscribe(channel string, handler func(message []byte)) error {
	conn := redisPool.Get()
	defer conn.Close()
	psc := redis.PubSubConn{Conn: conn}
	err := psc.Subscribe(channel)
	if err != nil {
		return err
	}
	for {
		switch v := psc.Receive().(type) {
		case redis.Message:
			go handler(v.Data)
		case redis.Subscription:
			logger.Infof("redis subscribe:", v.Channel, v.Count)
		case error:
			logger.Warnf("redis receive error:", v)
			return v
		}
	}
}

// Unlink 异步删除
func Unlink(key ...string) error {
	conn := redisPool.Get()
	defer conn.Close()
	args := []string{}
	for _, f := range key {
		args = append(args, f)
	}
	_, err := conn.Do("unlink", args)
	return err
}

// Del 删除Key
func Del(key string) error {
	conn := redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("del", key)
	return err
}

// Dels 删除Keys
func Dels(key ...string) error {
	conn := redisPool.Get()
	defer conn.Close()
	args := []string{}
	for _, f := range key {
		args = append(args, f)
	}
	_, err := conn.Do("del", args)
	return err
}

/**=========String===========**/

// Set string类型添加
func Set(name string, s string) error {
	conn := redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("SET", name, s)
	return err
}

func SetExp(name string, s string, exp time.Duration) error {
	conn := redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("SET", name, s)
	if err == nil && exp > 0 {
		conn.Do("expire", name, exp)
	}
	return err
}

// Get 获取字符值
func Get(name string) (string, error) {
	conn := redisPool.Get()
	defer conn.Close()
	tmp, err := redis.String(conn.Do("GET", name))
	if err != nil {
		return "", err
	}
	return tmp, err
}

// Expire 过期设置
func Expire(key string, exp time.Duration) error {
	conn := redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("EXPIRE", key, exp)
	return err
}

// Exist 判断指定的key是否存在
func Exist(name string) (bool, error) {
	conn := redisPool.Get()
	defer conn.Close()
	v, err := redis.Bool(conn.Do("EXISTS", name))
	return v, err
}

/**============Set==============**/

// SAdd 添加集合元素
func SAdd(setname string, value string, exp time.Duration) error {
	conn := redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("SADD", setname, value)
	if err == nil && exp > 0 {
		conn.Do("EXPIRE", setname, exp)
	}
	return err
}

// SExists 判断给定集合中是否包含某个元素
func SExists(setname string, value string) (bool, error) {
	conn := redisPool.Get()
	defer conn.Close()
	return redis.Bool(conn.Do("SISMEMBER", setname, value))
}

// SExistStr 集合是否包含字符串
func SExistStr(setname string, value string) (bool, error) {
	conn := redisPool.Get()
	defer conn.Close()
	return redis.Bool(conn.Do("SISMEMBER", setname, value))
}

// SMembers 获取指定集合元素
func SMembers(setname string) ([]string, error) {
	conn := redisPool.Get()
	defer conn.Close()
	mems, err := redis.ByteSlices(conn.Do("SMEMBERS", setname))
	if err != nil {
		return nil, err
	}
	strs := []string{}
	for i := 0; i < len(mems); i++ {
		strs = append(strs, string(mems[i]))
	}
	return strs, err
}

// SRem 删除指定集合元素
func SRem(key string, value string) error {
	conn := redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("srem", key, value)
	return err
}

// SRange 返回集合随机元素数量
func SRange(key string, count int) ([]string, error) {
	conn := redisPool.Get()
	defer conn.Close()
	values := []string{}
	vals, err := redis.Values(conn.Do("SRANDMEMBER", key, count))
	if err != nil {
		return values, err
	}
	for _, v := range vals {
		values = append(values, string(v.([]byte)))
	}
	return values, err
}

// ScardInt64s 获取集合中元素的个数
func ScardInt64s(name string) (int64, error) {
	conn := redisPool.Get()
	defer conn.Close()
	v, err := redis.Int64(conn.Do("SCARD", name))
	return v, err
}

/**==========Hash=============**/

// HGet 获取单个hash的值
func HGet(name, field string) (string, error) {
	conn := redisPool.Get()
	defer conn.Close()
	tmp, err := redis.String(conn.Do("HGET", name, field))
	if err != nil {
		return "", err
	}
	return tmp, err
}

// HSet 设置单个值，value可以为hash，列表,struct等
func HSet(hash, key string, value string, exp time.Duration) error {
	conn := redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("HSET", hash, key, value)
	if err == nil && exp > 0 {
		conn.Do("EXPIRE", hash, exp)
	}
	return err
}

// HGets 获取指定hash字段对应的多值
func HGets(hash string, keys ...string) ([]interface{}, error) {
	conn := redisPool.Get()
	defer conn.Close()
	args := []interface{}{hash}
	for _, f := range keys {
		args = append(args, f)
	}
	vals, err := redis.Values(conn.Do("HMGET", args))
	return vals, err
}

// HLen hash字典建的个数
func HLen(hash string) (int, error) {
	conn := redisPool.Get()
	defer conn.Close()
	return redis.Int(conn.Do("HLEN", hash))
}

// HExists 查看hash字典中指定的字段是否存在
func HExists(hash, key string) (bool, error) {
	conn := redisPool.Get()
	defer conn.Close()
	return redis.Bool(conn.Do("HEXISTS", hash, key))
}

// HDel 删除指定hash(字典)的key
func HDel(hash, key string) (bool, error) {
	conn := redisPool.Get()
	defer conn.Close()
	v, err := redis.Bool(conn.Do("HDEL", hash, key))
	return v, err
}

// LRange 列表范围
func LRange(key string, start, end int) ([]string, error) {
	conn := redisPool.Get()
	defer conn.Close()
	var values []string
	vals, err := redis.Values(conn.Do("lrange", key, start, end))
	if err != nil {
		return values, err
	}
	for _, v := range vals {
		values = append(values, string(v.([]byte)))
	}
	return values, nil
}

// LRem  删除指定元素
// count > 0 : 从表头开始向表尾搜索，移除与 VALUE 相等的元素，数量为 COUNT 。
// count < 0 : 从表尾开始向表头搜索，移除与 VALUE 相等的元素，数量为 COUNT 的绝对值。
// count = 0 : 移除表中所有与 VALUE 相等的值。
func LRem(key string, count int, val string) error {
	conn := redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("lrem", key, count, val)
	return err
}

// LPush 列表
func LPush(key string, data string) error {
	conn := redisPool.Get()
	defer conn.Close()
	_, err := conn.Do("lpush", key, data)
	return err
}

// LPop list
func LPop(key string) (interface{}, error) {
	conn := redisPool.Get()
	defer conn.Close()
	ret, err := conn.Do("lpop", key)
	return ret, err
}

// RPop list
func RPop(key string) (interface{}, error) {
	conn := redisPool.Get()
	defer conn.Close()
	ret, err := conn.Do("rpop", key)
	return ret, err
}

// BRPop 阻塞list
func BRPop(key string, timeout int) (string, error) {
	conn := redisPool.Get()
	defer conn.Close()
	ret, err := redis.Strings(conn.Do("brpop", key, timeout))
	if err != nil {
		return "", err
	}
	if len(ret) > 1 {
		return ret[1], nil
	}
	return "", fmt.Errorf("读取(%v)异常", key)
}
