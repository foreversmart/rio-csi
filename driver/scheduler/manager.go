package scheduler

import (
	"github.com/container-storage-interface/spec/lib/go/csi"
	"qiniu.io/rio-csi/driver/dparams"
	"sync"
)

type Manager struct {
	SchedulerMap map[string]*VolumeScheduler
	Lock         sync.Mutex
}

func NewManager() *Manager {
	return &Manager{
		SchedulerMap: make(map[string]*VolumeScheduler),
	}
}

// ScheduleVolume volume to a specific node
// TODO support multi Vg allocate
func (m *Manager) ScheduleVolume(req *csi.CreateVolumeRequest, params *dparams.VolumeParams) (nodeName string, err error) {
	vgPatternStr := params.VgPattern.String()

	m.Lock.Lock()
	scheduler, ok := m.SchedulerMap[vgPatternStr]
	if !ok {
		scheduler, err = NewVolumeScheduler(vgPatternStr)
		if err != nil {
			m.Lock.Unlock()
			return "", err
		}

		m.SchedulerMap[vgPatternStr] = scheduler
		m.Lock.Unlock()
	}

	return scheduler.ScheduleVolume(req)
}
