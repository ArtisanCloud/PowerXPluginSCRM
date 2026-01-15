package integration

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Job 描述一个可周期执行的后台任务。
type Job interface {
	Name() string
	Interval() time.Duration
	Run(ctx context.Context) error
}

// JobFunc 便于通过函数快速声明任务。
type JobFunc struct {
	name     string
	interval time.Duration
	fn       func(ctx context.Context) error
}

// NewJobFunc 构造一个基于函数的 Job。
func NewJobFunc(name string, interval time.Duration, fn func(ctx context.Context) error) JobFunc {
	return JobFunc{name: name, interval: interval, fn: fn}
}

// Name 返回任务名称。
func (j JobFunc) Name() string { return j.name }

// Interval 返回执行间隔。
func (j JobFunc) Interval() time.Duration { return j.interval }

// Run 调用任务函数。
func (j JobFunc) Run(ctx context.Context) error {
	if j.fn == nil {
		return nil
	}
	return j.fn(ctx)
}

// Scheduler 负责调度 Integration 背景任务。
type Scheduler struct {
	logger *logrus.Entry

	mu      sync.Mutex
	jobs    []Job
	started bool
	stopCh  chan struct{}
	wg      sync.WaitGroup
}

// NewScheduler 构造 Scheduler。
func NewScheduler(logger *logrus.Entry) *Scheduler {
	if logger == nil {
		logger = logrus.WithField("component", "integration.scheduler")
	}
	return &Scheduler{
		logger: logger,
	}
}

// Register 添加新的后台任务（需在 Start 前调用）。
func (s *Scheduler) Register(job Job) {
	if job == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		s.logger.
			WithField("job", job.Name()).
			Warn("attempted to register job after scheduler start; ignoring")
		return
	}
	s.jobs = append(s.jobs, job)
}

// Start 启动所有已注册任务。
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return
	}
	s.started = true
	if s.stopCh == nil {
		s.stopCh = make(chan struct{})
	}
	jobs := append([]Job(nil), s.jobs...)
	s.mu.Unlock()

	for _, job := range jobs {
		s.wg.Add(1)
		go s.runJob(ctx, job)
	}
}

// Stop 停止所有任务并等待退出。
func (s *Scheduler) Stop(ctx context.Context) {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return
	}
	close(s.stopCh)
	s.started = false
	s.mu.Unlock()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		s.logger.WithError(ctx.Err()).Warn("integration scheduler stop timed out")
	}
}

func (s *Scheduler) runJob(ctx context.Context, job Job) {
	defer s.wg.Done()
	interval := job.Interval()
	if interval <= 0 {
		interval = time.Minute
	}

	logger := s.logger.WithField("job", job.Name())
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.execute(ctx, job, logger) // immediate run on start

	for {
		select {
		case <-ticker.C:
			s.execute(ctx, job, logger)
		case <-s.stopCh:
			logger.Debug("scheduler stop signal received")
			return
		case <-ctx.Done():
			logger.WithError(ctx.Err()).Debug("scheduler context cancelled")
			return
		}
	}
}

func (s *Scheduler) execute(ctx context.Context, job Job, logger *logrus.Entry) {
	defer func() {
		if r := recover(); r != nil {
			logger.WithField("panic", r).Error("integration job panicked")
		}
	}()

	start := time.Now()
	if err := job.Run(ctx); err != nil {
		logger.WithError(err).Error("integration job execution failed")
		return
	}
	logger.WithField("elapsed", time.Since(start)).Debug("integration job executed")
}
