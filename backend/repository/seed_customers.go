package repository

import (
	"context"

	"escalator/entity"
)

//顧客が1件も無いときだけ、サンプルの顧客を登録する
func SeedCustomersIfEmpty(ctx context.Context, repo CustomerRepository) error {
	customers, err := repo.List(ctx)
	if err != nil {
		return err
	}
	if len(customers) > 0 {
		return nil
	}

	samples := []struct {
		name string
		plan entity.Plan
	}{
		{name: "サンプル Free", plan: entity.PlanFree},
		{name: "サンプル Pro", plan: entity.PlanPro},
		{name: "サンプル Enterprise", plan: entity.PlanEnterprise},
	}
	for _, sample := range samples {
		slaMinutes, err := entity.DefaultSLAMinutes(sample.plan)
		if err != nil {
			return err
		}
		customer, err := entity.NewCustomer(sample.name, sample.plan, slaMinutes)
		if err != nil {
			return err
		}
		if _, err := repo.Save(ctx, customer); err != nil {
			return err
		}
	}
	return nil
}
