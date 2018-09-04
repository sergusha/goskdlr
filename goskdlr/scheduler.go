package goskdlr

import (
	"context"
	"fmt"
	"sync"
	"time"
)

//A Scheduler manages the schedule of user jobs with a given presicion
type Scheduler struct {
	mx sync.Mutex

	//scheduled but not running yet jobs
	jobsPending map[JobID]*Job

	jobsInProgress map[JobID]*Job
	jobScheduled   chan *Job

	ctx    context.Context
	cancel context.CancelFunc

	clock *time.Ticker

	resume chan struct{}
	quit   chan struct{}

	//jobErrors chan error
}

func New(updatePeriod string) (*Scheduler, error) {
	d, err := time.ParseDuration(updatePeriod)

	if err != nil {
		return nil, fmt.Errorf("Error parsing update period: %v", err)
	}

	t := time.NewTicker(d)

	s := &Scheduler{
		resume:         make(chan struct{}),
		quit:           make(chan struct{}),
		jobsPending:    make(map[JobID]*Job),
		jobsInProgress: make(map[JobID]*Job),
		jobScheduled:   make(chan *Job),
		clock:          t,
	}

	s.ctx, s.cancel = context.WithCancel(context.Background())
	go s.run()

	return s, nil
}

func (s *Scheduler) runClock() {
	for {
		select {
		case now := <-s.clock.C:
			jobsToRemove := make([]JobID, 0)
			for jobID, job := range s.jobsPending {
				if job.startAt == nil {
					jobsToRemove = append(jobsToRemove, jobID)
				} else if job.startAt.Before(now) {
					s.jobScheduled <- job
					job.CalculateNextStartTime()
				}
			}
			for _, jobID := range jobsToRemove {
				s.deletePendingJob(jobID)
			}
		case <-s.quit:
			s.clock.Stop()
			s.quit <- struct{}{}
		}
	}
}

func (s *Scheduler) deletePendingJob(id JobID) {
	s.mx.Lock()
	defer s.mx.Unlock()
	delete(s.jobsPending, id)
}

func (s *Scheduler) CancelJob(job *Job) {
	s.mx.Lock()
	defer s.mx.Unlock()
	if _, ok := s.jobsPending[job.id]; ok {
		delete(s.jobsPending, job.id)
	} else if v, ok := s.jobsInProgress[job.id]; ok {
		v.cancel()
	}
	delete(s.jobsInProgress, job.id)

}

func (s *Scheduler) startJob(job *Job) {
	s.mx.Lock()
	defer s.mx.Unlock()

	go job.run()

}

// Stop stops the scheduler and cancels all pending jobs.
func (s *Scheduler) Stop() {
	s.mx.Lock()
	s.cancel()
	s.cancel = nil
	s.quit <- struct{}{}
	<-s.quit
	s.mx.Unlock()
}

func (s *Scheduler) ScheduleJob(job *Job) error {
	s.mx.Lock()
	defer s.mx.Unlock()

	if s.cancel == nil {
		return fmt.Errorf("ScheduleJob: schedule to a stopped scheduler")
	}
	if s.jobsPending == nil {
		return fmt.Errorf("ScheduleJob: jobsPending equals nil")
	}
	//resume the scheduler
	if len(s.jobsPending) == 0 {
		select {
		case s.resume <- struct{}{}:
		default:
		}
	}

	s.jobsPending[job.id] = job

	return nil
}

func (s *Scheduler) run() {
	defer func() {
		close(s.quit)
		close(s.resume)
	}()

	go s.runClock()

	for {
		select {
		case currentJob := <-s.jobScheduled:
			s.mx.Lock()
			go s.startJob(currentJob)
			s.jobsInProgress[currentJob.id] = currentJob
			s.mx.Unlock()
		case <-s.ctx.Done():
			s.mx.Lock()
			defer s.mx.Unlock()
			jobsToCancel := s.jobsInProgress
			s.jobsInProgress = nil
			s.jobsPending = nil
			for _, job := range jobsToCancel {
				job.cancel()
			}
			<-s.quit
			return
		}

	}
}
