package usecase

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"sync"
	"time"

	"escalator/entity"
	"escalator/repository"
)

type RunDemo struct {
	users     repository.UserRepository
	customers repository.CustomerRepository
	demos     repository.DemoRepository
	create    *CreateTicket
	notices   Notifier
	base      context.Context
	mu        sync.Mutex
	cancels   map[string]context.CancelFunc
}

func NewRunDemo(users repository.UserRepository, customers repository.CustomerRepository, demos repository.DemoRepository, create *CreateTicket, notices Notifier, base context.Context) *RunDemo {
	if base == nil {
		base = context.Background()
	}
	return &RunDemo{
		users:     users,
		customers: customers,
		demos:     demos,
		create:    create,
		notices:   notices,
		base:      base,
		cancels:   make(map[string]context.CancelFunc),
	}
}

//前回の起動で残った実行中のデモを止める
func (d *RunDemo) Recover(ctx context.Context) error {
	return d.demos.CloseInterrupted(ctx, time.Now())
}

//管理者がデモを始め、間隔をあけて通常の起票と同じ道で問い合わせを作る
func (d *RunDemo) Start(ctx context.Context, authorization string, count, intervalSec int, distribution string) (entity.DemoRun, error) {
	actor, err := d.create.current.Execute(ctx, authorization)
	if err != nil {
		return entity.DemoRun{}, err
	}
	if actor.Role != entity.RoleAdmin {
		return entity.DemoRun{}, entity.ErrForbidden
	}
	if err := entity.ValidateDemo(count, intervalSec, distribution); err != nil {
		return entity.DemoRun{}, err
	}
	applicants, err := d.users.ListApplicants(ctx)
	if err != nil {
		return entity.DemoRun{}, err
	}
	customers, err := d.customers.List(ctx)
	if err != nil {
		return entity.DemoRun{}, err
	}
	grouped := applicantsByCustomer(applicants)
	eligible := eligibleCustomers(customers, grouped)
	if len(eligible) == 0 {
		return entity.DemoRun{}, entity.ErrDemoNeedsApplicant
	}
	run, err := d.demos.Start(ctx, entity.DemoRun{
		StartedBy:    actor.ID,
		TotalCount:   count,
		IntervalSec:  intervalSec,
		Distribution: distribution,
		StartedAt:    time.Now(),
	})
	if err != nil {
		return entity.DemoRun{}, err
	}
	loopCtx, cancel := context.WithCancel(d.base)
	d.track(run.ID, cancel)
	go d.generate(loopCtx, cancel, run, eligible, grouped)
	return run, nil
}

//実行中のデモを止めるすでに作った問い合わせはそのまま残す
func (d *RunDemo) Stop(ctx context.Context, authorization string) (entity.DemoRun, error) {
	actor, err := d.create.current.Execute(ctx, authorization)
	if err != nil {
		return entity.DemoRun{}, err
	}
	if actor.Role != entity.RoleAdmin {
		return entity.DemoRun{}, entity.ErrForbidden
	}
	run, err := d.demos.StopRunning(ctx, time.Now())
	if err != nil {
		return entity.DemoRun{}, err
	}
	d.cancelRun(run.ID)
	d.notify(run)
	return run, nil
}

func (d *RunDemo) generate(ctx context.Context, cancel context.CancelFunc, run entity.DemoRun, customers []entity.Customer, grouped map[string][]entity.User) {
	defer cancel()
	defer d.untrack(run.ID)
	for i := 0; i < run.TotalCount; i++ {
		if ctx.Err() != nil {
			d.finish(run.ID, entity.DemoStopped)
			return
		}
		customer := pickCustomer(run.Distribution, customers)
		people := grouped[customer.ID]
		applicant := people[rand.IntN(len(people))]
		severity := pickSeverity(run.Distribution)
		_, err := d.create.CreateFor(
			context.Background(),
			applicant.ID,
			customer,
			fmt.Sprintf("デモの問い合わせ %d", i+1),
			"デモで作った問い合わせです。待ち順には、通常の起票と同じように載ります。",
			severity,
			pickCategory(run.Distribution, severity),
		)
		if err != nil {
			log.Printf("デモの問い合わせを作れませんでした。続きは作ります。%d件目: %v", i+1, err)
		} else if updated, err := d.demos.AddGenerated(context.Background(), run.ID); err != nil {
			if errors.Is(err, entity.ErrDemoNotRunning) {
				return
			}
			log.Printf("デモで作った件数を保存できませんでした。問い合わせは作ってあります。%d件目: %v", i+1, err)
		} else {
			d.notify(updated)
		}
		if i == run.TotalCount-1 {
			break
		}
		timer := time.NewTimer(time.Duration(run.IntervalSec) * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			d.finish(run.ID, entity.DemoStopped)
			return
		case <-timer.C:
		}
	}
	d.finish(run.ID, entity.DemoCompleted)
}

func (d *RunDemo) finish(id, status string) {
	run, changed, err := d.demos.Finish(context.Background(), id, status, time.Now())
	if err != nil {
		log.Printf("デモの終わりを保存できませんでした。id=%s: %v", id, err)
		return
	}
	if changed {
		d.notify(run)
	}
}

func (d *RunDemo) notify(run entity.DemoRun) {
	publishUser(d.notices, run.StartedBy, map[string]any{
		"type":             "demo_progress",
		"generated_count":  run.GeneratedCount,
		"total_count":      run.TotalCount,
		"status":           run.Status,
		"interval_seconds": run.IntervalSec,
		"distribution":     run.Distribution,
	})
}

func (d *RunDemo) track(id string, cancel context.CancelFunc) {
	d.mu.Lock()
	d.cancels[id] = cancel
	d.mu.Unlock()
}

func (d *RunDemo) cancelRun(id string) {
	d.mu.Lock()
	cancel := d.cancels[id]
	d.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (d *RunDemo) untrack(id string) {
	d.mu.Lock()
	delete(d.cancels, id)
	d.mu.Unlock()
}

func applicantsByCustomer(applicants []entity.User) map[string][]entity.User {
	grouped := make(map[string][]entity.User)
	for _, applicant := range applicants {
		if applicant.CustomerID == "" {
			continue
		}
		grouped[applicant.CustomerID] = append(grouped[applicant.CustomerID], applicant)
	}
	return grouped
}

func eligibleCustomers(customers []entity.Customer, grouped map[string][]entity.User) []entity.Customer {
	eligible := make([]entity.Customer, 0, len(customers))
	for _, customer := range customers {
		if len(grouped[customer.ID]) > 0 {
			eligible = append(eligible, customer)
		}
	}
	return eligible
}

func pickCustomer(distribution string, customers []entity.Customer) entity.Customer {
	total := 0
	weights := make([]int, len(customers))
	for i, customer := range customers {
		weights[i] = customerWeight(distribution, customer.Plan)
		total += weights[i]
	}
	roll := rand.IntN(total)
	for i, weight := range weights {
		roll -= weight
		if roll < 0 {
			return customers[i]
		}
	}
	return customers[len(customers)-1]
}

func customerWeight(distribution string, plan entity.Plan) int {
	if distribution != entity.DemoIncident {
		return 1
	}
	switch plan {
	case entity.PlanEnterprise:
		return 4
	case entity.PlanPro:
		return 2
	default:
		return 1
	}
}

func pickSeverity(distribution string) int {
	if distribution != entity.DemoIncident {
		return rand.IntN(4) + 1
	}
	roll := rand.IntN(15)
	switch {
	case roll < 1:
		return 1
	case roll < 3:
		return 2
	case roll < 7:
		return 3
	default:
		return 4
	}
}

func pickCategory(distribution string, severity int) entity.Category {
	if distribution == entity.DemoIncident {
		switch {
		case severity >= 3:
			return entity.CategoryIncident
		case severity == 2:
			return entity.CategoryBug
		default:
			return entity.CategoryQuestion
		}
	}
	categories := []entity.Category{
		entity.CategoryIncident,
		entity.CategoryBug,
		entity.CategoryQuestion,
		entity.CategoryRequest,
		entity.CategoryOther,
	}
	return categories[rand.IntN(len(categories))]
}
