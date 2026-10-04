package services

import (
	"context"
	"fmt"
	"github.com/yourusername/saga-service/internal/domains"
	"github.com/yourusername/saga-service/internal/repositories/postgres"
	"github.com/yourusername/saga-service/pkg/logger"
	"time"
)

type ResumeSagaWorker struct {
	tasks    chan *domains.OrderSaga
	ctx      context.Context
	cancel   context.CancelFunc
	nWorkers int

	sagaService *OrderSagaService
	sagaRepo    postgres.CreateOrderSagaRepoInt
}

func NewResumeSagaWorker(workers int, sagaService *OrderSagaService, sagaRepo postgres.CreateOrderSagaRepoInt) *ResumeSagaWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &ResumeSagaWorker{
		tasks:    make(chan *domains.OrderSaga, 10),
		ctx:      ctx,
		cancel:   cancel,
		nWorkers: workers,

		sagaService: sagaService,
		sagaRepo:    sagaRepo,
	}
}

func (r *ResumeSagaWorker) Run() {
	go r.producer()

	for i := 0; i < r.nWorkers; i++ {
		go r.consumer()
	}
}

func (r *ResumeSagaWorker) consumer() {
	for {
		task, ok := <-r.tasks
		if !ok {
			return
		}

		ctx := context.WithValue(context.Background(), "saga", *task)
		if task.Cancelled {
			r.sagaService.ContinueCancel(ctx)
		} else {
			r.sagaService.Continue(ctx)
		}
	}
}

func (r *ResumeSagaWorker) producer() {
	place := "[ResumeSagaWorker.producer]"
	defer close(r.tasks)

	for {
		sagas, err := r.sagaRepo.GetUnfinishedSaga(r.ctx)
		if err != nil {
			logger.Log.Error(fmt.Sprintf("%s Не удалось получить незавершенные саги", place))
		}

		for _, saga := range sagas {
			select {
			case r.tasks <- saga:
			case <-r.ctx.Done():
				return
			}
		}

		select {
		case <-r.ctx.Done():
			return
		case <-time.After(time.Second * 5):
		}
	}
}

func (r *ResumeSagaWorker) Stop() {
	r.cancel()
}
