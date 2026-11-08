package seed

import (
	"context"
	"fmt"
	"log"

	"golang.org/x/crypto/bcrypt"

	"escalator/entity"
	"escalator/repository"
)

//利用者が1人も無いときだけ、サンプルの管理者、担当者、申請者を登録する
func SeedUsersIfEmpty(ctx context.Context, users repository.UserRepository, customers repository.CustomerRepository, password string, cost int) error {
	count, err := users.Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	if password == "" {
		log.Printf("サンプル利用者のパスワードが無いため、利用者は入れませんでした")
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return err
	}
	if err := saveStaff(ctx, users, string(hash), password); err != nil {
		return err
	}
	listed, err := customers.List(ctx)
	if err != nil {
		return err
	}
	return saveApplicants(ctx, users, listed, string(hash), password)
}

func saveStaff(ctx context.Context, users repository.UserRepository, hash, password string) error {
	admin, err := entity.NewAdmin("サンプル管理者", "admin@example.com", password)
	if err != nil {
		return err
	}
	admin.PasswordHash = hash
	if _, err := users.Save(ctx, admin); err != nil {
		return err
	}
	for _, sample := range []struct{ name, email string }{
		{name: "サンプル担当者1", email: "agent1@example.com"},
		{name: "サンプル担当者2", email: "agent2@example.com"},
	} {
		agent, err := entity.NewAgent(sample.name, sample.email, password)
		if err != nil {
			return err
		}
		agent.PasswordHash = hash
		if _, err := users.SaveAgent(ctx, agent); err != nil {
			return err
		}
	}
	return nil
}

func saveApplicants(ctx context.Context, users repository.UserRepository, customers []entity.Customer, hash, password string) error {
	samples := []struct {
		name  string
		email string
		plan  entity.Plan
		label string
	}{
		{name: "サンプル申請者 Free", email: "free@example.com", plan: entity.PlanFree, label: "無償"},
		{name: "サンプル申請者 Pro", email: "pro@example.com", plan: entity.PlanPro, label: "有償"},
		{name: "サンプル申請者 Enterprise", email: "enterprise@example.com", plan: entity.PlanEnterprise, label: "VIP"},
	}
	for _, sample := range samples {
		customer, err := customerForPlan(customers, sample.plan, sample.label)
		if err != nil {
			return err
		}
		applicant, err := entity.NewApplicant(sample.name, sample.email, password)
		if err != nil {
			return err
		}
		applicant.CustomerID = customer.ID
		applicant.PasswordHash = hash
		if _, err := users.Save(ctx, applicant); err != nil {
			return err
		}
	}
	log.Printf("サンプルの利用者を6人登録しました")
	return nil
}

func customerForPlan(customers []entity.Customer, plan entity.Plan, label string) (entity.Customer, error) {
	var fallback entity.Customer
	foundFallback := false
	for _, customer := range customers {
		if customer.Plan != plan {
			continue
		}
		if !foundFallback {
			fallback = customer
			foundFallback = true
		}
		if customer.Name == "サンプル "+planName(plan) {
			return customer, nil
		}
	}
	if foundFallback {
		return fallback, nil
	}
	return entity.Customer{}, fmt.Errorf("%sの顧客が無いため、サンプルの申請者を登録できませんでした", label)
}

func planName(plan entity.Plan) string {
	switch plan {
	case entity.PlanFree:
		return "Free"
	case entity.PlanPro:
		return "Pro"
	case entity.PlanEnterprise:
		return "Enterprise"
	default:
		return ""
	}
}
