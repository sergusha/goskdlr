package goskdlr

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type JobID uint64

var jobCounter JobID
var jobCountMx sync.Mutex

type Job struct {
	id      JobID
	startAt *time.Time
	period  *time.Duration

	ctx    context.Context
	cancel context.CancelFunc

	returnValue chan interface{}

	//toCall function must return value or error via the context
	handler func(ctx context.Context)
}

func NewJob(ctx context.Context, startAt *time.Time, period *time.Duration, handler func(ctx context.Context)) (*Job, error) {
	//if startAt.Before(time.Now()) {
	//	return nil, fmt.Errorf("NewJob: Job can't be started in the past")
	//}
	jobCountMx.Lock()
	defer jobCountMx.Unlock()
	jobCounter += 1
	j := &Job{
		id:      jobCounter,
		startAt: startAt,
		period:  period,
		handler: handler,
	}
	j.ctx, j.cancel = context.WithCancel(ctx)
	return j, nil
}

func (j *Job) CalculateNextStartTime() error {
	if j.cancel == nil {
		return fmt.Errorf("CalculateNextStartTime: job is canceled")
	}
	if j.startAt == nil {
		return fmt.Errorf("CalculateNextStartTime: startAt equals nil")
	}
	if j.period == nil {
		//job is never called again
		j.startAt = nil
		return nil
	} else {
		j.startAt.Add(*j.period)
		return nil
	}
}

func (j *Job) run() {
	if j.cancel == nil {
		return
	}
	j.handler(j.ctx)

}
