package postgres

import (
	"context"

	"escalator/domain"
)

//顧客が1件も無いときだけ、サンプルの顧客を登録する
func SeedCustomersIfEmpty(ctx context.Context, repo *CustomerRepository) error {
	customers, err := repo.List(ctx)
	if err != nil {
		return err
	}
	if len(customers) > 0 {
		return nil
	}

	samples := []struct {
		name string
		plan domain.Plan
	}{
		{name: "サンプル Free", plan: domain.PlanFree},
		{name: "サンプル Pro", plan: domain.PlanPro},
		{name: "サンプル Enterprise", plan: domain.PlanEnterprise},
	}
	for _, sample := range samples {
		slaMinutes, err := domain.DefaultSLAMinutes(sample.plan)
		if err != nil {
			return err
		}
		customer, err := domain.NewCustomer(sample.name, sample.plan, slaMinutes)
		if err != nil {
			return err
		}
		if _, err := repo.Save(ctx, customer); err != nil {
			return err
		}
	}
	return nil
}
