package pkg

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/log"
	"fmt"
	"github.com/spf13/viper"
)

type Config struct {
	PromeServers     map[string]string `yaml:"prome_servers"`
	ReplaceLabelName string            `yaml:"replace_label_name"`
	Http             *Http             `yaml:"http"`
}

type Http struct {
	ListenAddr string `yaml:"listen_addr"`
}

var (
	RouteLabelRegStr string
	RouteLabelRegStrP string
	QueryPlaceRegStr string
	RromeServerMap map[string]string
)

func InitLabelRegStr(labelName string) {
	RouteLabelRegStr = fmt.Sprintf(`%s="(?P<%s>.*?)"`, labelName, labelName)
	RouteLabelRegStrP = fmt.Sprintf(`%s=~"(?P<%s>.*?)"`, labelName, labelName)
	QueryPlaceRegStr = `{,`
}



func Load(s string) (*Config, error) {
	cfg := &Config{}

	err := yaml.UnmarshalStrict([]byte(s), cfg)

	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return cfg, nil
}

func LoadFile(filename string, logger log.Logger) (*Config, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	cfg, err := Load(string(content))
	if err != nil {
		level.Error(logger).Log("msg", "parsing YAML file errr...", "error", err)
		return nil, err
	}

	return cfg, nil
}

func SetDefaultVar(sc *Config) {
	if sc.ReplaceLabelName != "" {
		viper.SetDefault("replace_label_name", sc.ReplaceLabelName)
	} else {
		viper.SetDefault("replace_label_name", "cluster")
	}
}
