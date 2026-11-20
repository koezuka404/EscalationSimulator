package rule

import (
	"unicode/utf8"

	"escalator/entity"
)

//名前、プラン、目標時間を確かめて顧客を作る
func NewCustomer(name string, plan entity.Plan, slaMinutes int) (entity.Customer, error) {
	if utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 100 {
		return entity.Customer{}, ErrInvalidCustomerName
	}
	if slaMinutes < 1 || slaMinutes > 1440 {
		return entity.Customer{}, ErrInvalidSLAMinutes
	}
	if _, err := PlanScore(entity.Customer{Plan: plan}); err != nil {
		return entity.Customer{}, err
	}
	return entity.Customer{Name: name, Plan: plan, SLAMinutes: slaMinutes}, nil
}

//プランごとの初期の目標時間を返す
func DefaultSLAMinutes(plan entity.Plan) (int, error) {
	switch plan {
	case entity.PlanFree:
		return 120, nil
	case entity.PlanPro:
		return 60, nil
	case entity.PlanEnterprise:
		return 15, nil
	default:
		return 0, ErrInvalidPlan
	}
}

//プランを点数にするFreeは1、Proは2、Enterpriseは3
func PlanScore(customer entity.Customer) (int, error) {
	switch customer.Plan {
	case entity.PlanFree:
		return 1, nil
	case entity.PlanPro:
		return 2, nil
	case entity.PlanEnterprise:
		return 3, nil
	default:
		return 0, ErrInvalidPlan
	}
}
