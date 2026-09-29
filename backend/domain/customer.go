package domain

import (
	"errors"
	"unicode/utf8"
)

type Plan string

const (
	PlanFree       Plan = "free"
	PlanPro        Plan = "pro"
	PlanEnterprise Plan = "enterprise"
)

var (
	ErrInvalidCustomerName = errors.New("顧客名は1〜100文字です")
	ErrInvalidPlan         = errors.New("プランは Free、Pro、Enterprise のいずれかです")
	ErrInvalidSLAMinutes   = errors.New("目標時間は1〜1440分です")
)

// Customer は社外の契約先。保存先は知らない。
type Customer struct {
	ID         string
	Name       string
	Plan       Plan
	SLAMinutes int
}

func NewCustomer(name string, plan Plan, slaMinutes int) (Customer, error) {
	if utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 100 {
		return Customer{}, ErrInvalidCustomerName
	}
	if slaMinutes < 1 || slaMinutes > 1440 {
		return Customer{}, ErrInvalidSLAMinutes
	}
	if _, err := planScore(plan); err != nil {
		return Customer{}, err
	}
	return Customer{Name: name, Plan: plan, SLAMinutes: slaMinutes}, nil
}

func DefaultSLAMinutes(plan Plan) (int, error) {
	switch plan {
	case PlanFree:
		return 120, nil
	case PlanPro:
		return 60, nil
	case PlanEnterprise:
		return 15, nil
	default:
		return 0, ErrInvalidPlan
	}
}

// PlanScore は点数計算用のプラン点数を返す。
func (c Customer) PlanScore() (int, error) {
	return planScore(c.Plan)
}

func planScore(plan Plan) (int, error) {
	switch plan {
	case PlanFree:
		return 1, nil
	case PlanPro:
		return 2, nil
	case PlanEnterprise:
		return 3, nil
	default:
		return 0, ErrInvalidPlan
	}
}
