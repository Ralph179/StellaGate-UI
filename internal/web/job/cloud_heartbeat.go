package job

import "github.com/mhsanaei/3x-ui/v3/internal/web/service"

type CloudHeartbeatJob struct {
	activation service.StellaLocalActivationService
}

func NewCloudHeartbeatJob() *CloudHeartbeatJob {
	return &CloudHeartbeatJob{}
}

func (j *CloudHeartbeatJob) Run() {
	_ = j.activation.Heartbeat()
}
