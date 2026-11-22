package rule

import (
	"testing"
	"time"

	"escalator/entity"
)

func TestPriorityDecisionTable(t *testing.T) {
	now := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	cases := []struct {
		name   string
		a, b   sample
		firstA bool
	}{
		{
			name:   "緊急度が違う",
			a:      sample{severity: 4, plan: entity.PlanFree, wait: 0},
			b:      sample{severity: 3, plan: entity.PlanEnterprise, wait: 99},
			firstA: true,
		},
		{
			name:   "対応期限切れのFreeは時間内のEnterpriseより先",
			a:      sample{severity: 2, plan: entity.PlanFree, wait: 120},
			b:      sample{severity: 2, plan: entity.PlanEnterprise, wait: 0},
			firstA: true,
		},
		{
			name:   "同じ緊急度で時間内ならプランが高い方",
			a:      sample{severity: 3, plan: entity.PlanEnterprise, wait: 1},
			b:      sample{severity: 3, plan: entity.PlanFree, wait: 10},
			firstA: true,
		},
		{
			name:   "どちらも対応期限切れならプランが高い方",
			a:      sample{severity: 2, plan: entity.PlanPro, wait: 200},
			b:      sample{severity: 2, plan: entity.PlanFree, wait: 200},
			firstA: true,
		},
		{
			name:   "条件が同じなら待ちが長い方",
			a:      sample{severity: 1, plan: entity.PlanPro, wait: 50},
			b:      sample{severity: 1, plan: entity.PlanPro, wait: 10},
			firstA: true,
		},
		{
			name:   "Freeの全停止はEnterpriseの質問より先",
			a:      sample{severity: 4, plan: entity.PlanFree, wait: 0},
			b:      sample{severity: 1, plan: entity.PlanEnterprise, wait: 0},
			firstA: true,
		},
		{
			name:   "時間内の待ち99分はプランの差を追い越さない",
			a:      sample{severity: 2, plan: entity.PlanFree, wait: 99},
			b:      sample{severity: 2, plan: entity.PlanEnterprise, wait: 0},
			firstA: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			scoreA, createdA := tc.a.score(t, now)
			scoreB, createdB := tc.b.score(t, now)
			got := WaitingFirst(scoreA, createdA, "ticket-b", scoreB, createdB, "ticket-a")
			if got != tc.firstA {
				t.Fatalf("先が違う A=%d B=%d 先A=%v", scoreA, scoreB, got)
			}
		})
	}
}

func TestSameScoreOrder(t *testing.T) {
	earlier := time.Date(2026, 11, 23, 9, 0, 0, 0, time.UTC)
	later := earlier.Add(time.Minute)
	if !WaitingFirst(10, earlier, "ticket-z", 10, later, "ticket-a") {
		t.Fatal("点数が同じなら、先に作られた方を先にする")
	}
	if WaitingFirst(10, later, "ticket-a", 10, earlier, "ticket-z") {
		t.Fatal("後から作られた方が先になっている")
	}
	if !SameScoreFirst(earlier, "ticket-a", earlier, "ticket-b") {
		t.Fatal("作った時刻が同じなら、IDが小さい方を先にする")
	}
	if SameScoreFirst(earlier, "ticket-b", earlier, "ticket-a") {
		t.Fatal("IDが大きい方が先になっている")
	}
}

type sample struct {
	severity int
	plan     entity.Plan
	wait     int
}

func (s sample) score(t *testing.T, now time.Time) (int, time.Time) {
	t.Helper()
	planScore, err := PlanScore(entity.Customer{Plan: s.plan})
	if err != nil {
		t.Fatal(err)
	}
	sla, err := DefaultSLAMinutes(s.plan)
	if err != nil {
		t.Fatal(err)
	}
	created := now.Add(-time.Duration(s.wait) * time.Minute)
	got, err := PriorityScore(s.severity, planScore, created, now, sla, false)
	if err != nil {
		t.Fatal(err)
	}
	return got, created
}
