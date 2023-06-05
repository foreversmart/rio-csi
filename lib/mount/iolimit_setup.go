package mount

import (
	"qiniu.io/rio-csi/conf"
	"qiniu.io/rio-csi/logger"
	"strconv"
	"strings"
	"sync"
)

var (
	ioLimitsEnabled  = false
	containerRuntime string
	rwlock           sync.RWMutex

	riopsPerGBMap    map[string]uint64
	wiopsPerGBMap    map[string]uint64
	baseIopsPerGBMap map[string]uint64
	maxIopsPerGBMap  map[string]uint64
	rbpsPerGBMap     map[string]uint64
	wbpsPerGBMap     map[string]uint64
	baseBpsPerGBMap  map[string]uint64
	maxBpsPerGBMap   map[string]uint64
)

// SetIORateLimits sets io limit rates for the volume group (prefixes) provided in config
func SetIORateLimits(config *conf.Config) {
	rwlock.Lock()
	defer rwlock.Unlock()

	ioLimitsEnabled = true
	containerRuntime = config.ContainerRuntime
	setValues(config)
}

func setValues(config *conf.Config) {
	var err error
	riopsVals := config.ReadIopsLimit
	wiopsVals := config.WriteIopsLimit
	baseIopsVals := config.BaseIopsLimit
	maxIopsVals := config.MaxIopsLimit

	rbpsVals := config.ReadBpsLimit
	wbpsVals := config.WriteBpsLimit
	baseBpsVals := config.BaseBpsLimit
	maxBpsVals := config.MaxBpsLimit

	riopsPerGBMap, err = extractRateValues(riopsVals)
	if err != nil {
		logger.StdLog.Warn("Read IOPS limit rates could not be extracted from config", err)
		riopsPerGBMap = map[string]uint64{}
	}

	wiopsPerGBMap, err = extractRateValues(wiopsVals)
	if err != nil {
		logger.StdLog.Warn("Write IOPS limit rates could not be extracted from config", err)
		wiopsPerGBMap = map[string]uint64{}
	}

	baseIopsPerGBMap, err = extractRateValues(baseIopsVals)
	if err != nil {
		logger.StdLog.Warn("base IOPS limit rates could not be extracted from config", err)
		baseIopsPerGBMap = map[string]uint64{}
	}

	maxIopsPerGBMap, err = extractRateValues(maxIopsVals)
	if err != nil {
		logger.StdLog.Warn("max IOPS limit rates could not be extracted from config", err)
		maxIopsPerGBMap = map[string]uint64{}
	}

	rbpsPerGBMap, err = extractRateValues(rbpsVals)
	if err != nil {
		logger.StdLog.Warn("Read BPS limit rates could not be extracted from config", err)
		rbpsPerGBMap = map[string]uint64{}
	}

	wbpsPerGBMap, err = extractRateValues(wbpsVals)
	if err != nil {
		logger.StdLog.Warn("Write BPS limit rates could not be extracted from config", err)
		wbpsPerGBMap = map[string]uint64{}
	}

	baseBpsPerGBMap, err = extractRateValues(baseBpsVals)
	if err != nil {
		logger.StdLog.Warn("base BPS limit rates could not be extracted from config", err)
		baseBpsPerGBMap = map[string]uint64{}
	}

	maxBpsPerGBMap, err = extractRateValues(maxBpsVals)
	if err != nil {
		logger.StdLog.Warn("max BPS limit rates could not be extracted from config", err)
		maxBpsPerGBMap = map[string]uint64{}
	}
}

func extractRateValues(rateVals *[]string) (map[string]uint64, error) {
	rate := map[string]uint64{}
	if rateVals == nil {
		return rate, nil
	}
	for _, kv := range *rateVals {
		parts := strings.Split(kv, "=")
		key := parts[0]
		value, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return nil, err
		}
		rate[key] = value
	}
	return rate, nil
}
