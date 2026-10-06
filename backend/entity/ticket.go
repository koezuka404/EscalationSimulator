package entity

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

type TicketStatus string

const (
	TicketOpen       TicketStatus = "open"
	TicketInProgress TicketStatus = "in_progress"
	TicketClosed     TicketStatus = "closed"
)

type Category string

const (
	CategoryIncident Category = "incident"
	CategoryBug      Category = "bug"
	CategoryQuestion Category = "question"
	CategoryRequest  Category = "request"
	CategoryOther    Category = "other"
)

var (
	ErrInvalidTitle        = errors.New("件名は1文字以上、200文字以内で入力してください")
	ErrInvalidDescription  = errors.New("詳細は1文字以上、5000文字以内で入力してください")
	ErrInvalidCategory     = errors.New("種類は障害、不具合、質問、要望、その他のどれかを選んでください")
	ErrTicketApplicant     = errors.New("チケットを起票できるのは申請者だけです")
	ErrCustomerRequired    = errors.New("所属顧客が設定されていません")
	ErrTicketNotFound      = errors.New("チケットが見つかりませんでした")
	ErrQueueForbidden      = errors.New("待ち順を見られるのは担当者と管理者だけです")
	ErrClaimAgent          = errors.New("次を引き取れるのは担当者だけです")
	ErrNotWaiting          = errors.New("待機中のときだけ引き取れます")
	ErrAgentBusy           = errors.New("対応中のチケットがあります")
	ErrQueueEmpty          = errors.New("待ちチケットがありません")
	ErrQueueUnavailable    = errors.New("待ち順を一時的に利用できません")
	ErrNotInProgress       = errors.New("対応中のチケットではありません")
	ErrCloseForbidden      = errors.New("このチケットを完了できるのは、担当者か管理者だけです")
	ErrInvalidCloseComment = errors.New("終了コメントは1文字以上、2000文字以内で入力してください")
)

type Ticket struct {
	ID            string
	CustomerID    string
	CreatedBy     string
	Title         string
	Description   string
	Severity      int
	Category      Category
	Status        TicketStatus
	AssigneeID    string
	PriorityScore int
	CreatedAt     time.Time
	ClaimedAt     time.Time
	ClosedAt      time.Time
	CloseComment  string
}

//件名、詳細、種類、緊急度を確かめて、対応待ちのチケットを作る
func NewTicket(customerID, createdBy, title, description string, severity int, category Category, createdAt time.Time) (Ticket, error) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	if customerID == "" {
		return Ticket{}, ErrCustomerRequired
	}
	if utf8.RuneCountInString(title) < 1 || utf8.RuneCountInString(title) > 200 {
		return Ticket{}, ErrInvalidTitle
	}
	if utf8.RuneCountInString(description) < 1 || utf8.RuneCountInString(description) > 5000 {
		return Ticket{}, ErrInvalidDescription
	}
	if severity < 1 || severity > 4 {
		return Ticket{}, ErrInvalidSeverity
	}
	if !ValidCategory(category) {
		return Ticket{}, ErrInvalidCategory
	}
	return Ticket{
		CustomerID:  customerID,
		CreatedBy:   createdBy,
		Title:       title,
		Description: description,
		Severity:    severity,
		Category:    category,
		Status:      TicketOpen,
		CreatedAt:   createdAt,
	}, nil
}

//終了コメントの文字数を確かめる
func CloseCommentText(comment string) (string, error) {
	comment = strings.TrimSpace(comment)
	if utf8.RuneCountInString(comment) < 1 || utf8.RuneCountInString(comment) > 2000 {
		return "", ErrInvalidCloseComment
	}
	return comment, nil
}

func ValidCategory(category Category) bool {
	switch category {
	case CategoryIncident, CategoryBug, CategoryQuestion, CategoryRequest, CategoryOther:
		return true
	default:
		return false
	}
}
