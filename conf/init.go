package conf

import (
	"context"
	"errors"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"qiniu.io/rio-csi/client"
	"qiniu.io/rio-csi/logger"
)

const (
	ConfigName = "riocsi-config"
)

func LoadConfig(namespace string) (driverConfig *Config, err error) {
	configmap, err := client.DefaultClient.ClientSet.CoreV1().ConfigMaps(namespace).Get(context.Background(), ConfigName, metav1.GetOptions{})
	if err != nil {
		logger.StdLog.Error(err)
		return nil, err
	}

	configStr, ok := configmap.Data["config.conf"]
	if !ok {
		return nil, errors.New("cant find config key config.conf in configmap")
	}

	err = yaml.Unmarshal([]byte(configStr), &driverConfig)
	return
}
