package conf

import (
	"context"
	"errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"qiniu.io/rio-csi/client"
	"qiniu.io/rio-csi/logger"
)

func LoadConfig(namespace string) (driverConfig *Config, err error) {
	configmap, err := client.DefaultClient.ClientSet.CoreV1().ConfigMaps(namespace).Get(context.Background(), "", metav1.GetOptions{})
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
