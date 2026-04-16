package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config 对应 config.json 的完整结构
type Config struct {
	App      AppConfig      `json:"app"`
	DB       DBConfig       `json:"db"`
	Redis    RedisConfig    `json:"redis"`
	RocketMQ RocketMQConfig `json:"rocketmq"`
	Session  SessionConfig  `json:"session"`
	Log      LogConfig      `json:"log"`
	OSS      OSSConfig      `json:"oss"`
}

type AppConfig struct {
	Name   string `json:"name"`
	NodeID string `json:"node_id"`
	Port   int    `json:"port"`
	Debug  bool   `json:"debug"` // true = 调试模式，记录请求日志
}

type DBConfig struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	User         string `json:"user"`
	Password     string `json:"password"`
	DBName       string `json:"dbname"`
	SSLMode      string `json:"sslmode"`
	MaxOpenConns int    `json:"max_open_conns"`
	MaxIdleConns int    `json:"max_idle_conns"`
}

type RedisConfig struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
	PoolSize int    `json:"pool_size"`
}

type RocketMQConfig struct {
	NameServer    string `json:"name_server"`
	ProducerGroup string `json:"producer_group"`
	RetryTimes    int    `json:"retry_times"`
}

type SessionConfig struct {
	TTL int `json:"ttl"` // 单位：秒
}

type LogConfig struct {
	Level  string `json:"level"`  // debug | info | warn | error
	Output string `json:"output"` // stdout | file
	Dir    string `json:"dir"`    // 日志目录，output=file 时生效，默认 logs
}

type OSSConfig struct {
	Endpoint  string `json:"endpoint"`
	Bucket    string `json:"bucket"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
}

// Load 从指定 JSON 文件加载配置
func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config file: %w", err)
	}
	defer f.Close()

	var cfg Config
	if err = json.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	// 设置默认值
	if cfg.Log.Dir == "" {
		cfg.Log.Dir = "logs"
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	if cfg.Log.Output == "" {
		cfg.Log.Output = "stdout"
	}

	return &cfg, nil
}
