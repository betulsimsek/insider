package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/useinsider/go-pkg/inslogger"
)

type SchedulerService interface {
	Start() error
	Stop() error
	IsRunning() bool
}

type schedulerService struct {
	logger       inslogger.Interface
	sender       MessageSender
	interval     time.Duration
	batchSize    int
	ticker       *time.Ticker
	stopChan     chan struct{}
	isRunning    bool
	runningMutex sync.Mutex
}

func NewSchedulerService(sender MessageSender, interval time.Duration, batchSize int) SchedulerService {
	return &schedulerService{
		sender:    sender,
		interval:  interval,
		batchSize: batchSize,
		stopChan:  make(chan struct{}),
	}
}

func (s *schedulerService) Start() error {
	s.logger.Log("Starting scheduler...")
	s.runningMutex.Lock()
	defer s.runningMutex.Unlock()

	if s.isRunning {
		return nil
	}

	s.ticker = time.NewTicker(s.interval)
	s.isRunning = true

	go func() {
		// Send messages immediately on start
		if err := s.sender.SendMessages(s.batchSize); err != nil {
			s.logger.Log(fmt.Errorf("error sending initial messages: %v", err))
		}

		for {
			select {
			case <-s.ticker.C:
				if err := s.sender.SendMessages(s.batchSize); err != nil {
					s.logger.Log(fmt.Errorf("error sending scheduled messages: %v", err))
				}
			case <-s.stopChan:
				s.ticker.Stop()
				return
			}
		}
	}()

	return nil
}

func (s *schedulerService) Stop() error {
	s.runningMutex.Lock()
	defer s.runningMutex.Unlock()

	if !s.isRunning {
		return nil
	}

	s.stopChan <- struct{}{}
	s.isRunning = false
	return nil
}

func (s *schedulerService) IsRunning() bool {
	s.runningMutex.Lock()
	defer s.runningMutex.Unlock()
	return s.isRunning
}
