package entity

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
	ErrInvalidCustomerName = errors.New("顧客名は1文字以上、100文字以内で入力してください")
	ErrInvalidPlan         = errors.New("プランは Free、Pro、Enterprise のどれかを選んでください")
	ErrInvalidSLAMinutes   = errors.New("初めて担当が付くまでの目標時間は、1分から24時間（1440分）の間で入力してください")
	ErrCustomerNotFound    = errors.New("指定した顧客は見つかりませんでした")
)

type Customer struct {
	ID         string
	Name       string
	Plan       Plan
	SLAMinutes int
}

//名前、プラン、目標時間を確かめて顧客を作る
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

//プランごとの初期の目標時間を返す
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

//プランを点数にするFreeは1、Proは2、Enterpriseは3
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
