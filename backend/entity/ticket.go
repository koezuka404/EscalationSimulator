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
	ErrInvalidTitle          = errors.New("件名は1文字以上、200文字以内で入力してください")
	ErrInvalidDescription    = errors.New("詳細は1文字以上、5000文字以内で入力してください")
	ErrInvalidCategory       = errors.New("種類は障害、不具合、質問、要望、その他のどれかを選んでください")
	ErrTicketApplicant       = errors.New("チケットを起票できるのは申請者だけです")
	ErrCustomerRequired      = errors.New("所属顧客が設定されていません")
	ErrTicketNotFound        = errors.New("チケットが見つかりませんでした")
	ErrQueueForbidden        = errors.New("待ち順を見られるのは担当者と管理者だけです")
	ErrClaimAgent            = errors.New("次を引き取れるのは担当者だけです")
	ErrNotWaiting            = errors.New("待機中のときだけ引き取れます")
	ErrAgentBusy             = errors.New("対応中のチケットがあります")
	ErrQueueEmpty            = errors.New("待ちチケットがありません")
	ErrQueueUnavailable      = errors.New("待ち順を一時的に利用できません")
	ErrNotInProgress         = errors.New("対応中のチケットではありません")
	ErrCloseForbidden        = errors.New("このチケットを完了できるのは、担当者か管理者だけです")
	ErrInvalidCloseComment   = errors.New("終了コメントは1文字以上、2000文字以内で入力してください")
	ErrInvalidSeverityReason = errors.New("理由は1文字以上、500文字以内で入力してください")
	ErrSeveritySame          = errors.New("緊急度は今と違う値を選んでください")
	ErrSeverityClosed        = errors.New("完了したチケットの緊急度は変えられません")
	ErrSeverityRole          = errors.New("緊急度を変えられるのは担当者と管理者だけです")
	ErrSeverityLower         = errors.New("緊急度を下げられるのは管理者だけです")
	ErrSeverityRaise         = errors.New("緊急度を上げられるのは、自分の対応中のチケットだけです")
	ErrReleaseForbidden      = errors.New("担当を外せるのは管理者だけです")
	ErrMyTicketsForbidden    = errors.New("自分のチケットを見られるのは申請者だけです")
	ErrTicketHidden          = errors.New("このチケットを見られるのは、担当した人と管理者だけです")
	ErrInvalidWorkNote       = errors.New("対応メモは1文字以上、5000文字以内で入力してください")
	ErrWorkNoteRole          = errors.New("対応メモを書けるのは担当者と管理者だけです")
	ErrWorkNoteAgent         = errors.New("対応メモを書けるのは、自分の対応中のチケットだけです")
	ErrWorkNoteClosed        = errors.New("完了したチケットには対応メモを書けません")
)

const WorkNoteKind = "work_note"

type WorkNote struct {
	ID        string
	TicketID  string
	UserID    string
	Body      string
	CreatedAt time.Time
}

type SeverityChange struct {
	FromSeverity int
	ToSeverity   int
	Reason       string
	ChangedBy    string
	CreatedAt    time.Time
}

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
	SlaNotifiedAt time.Time
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

//緊急度を変える理由の文字数を確かめる
func SeverityReason(reason string) (string, error) {
	reason = strings.TrimSpace(reason)
	if utf8.RuneCountInString(reason) < 1 || utf8.RuneCountInString(reason) > 500 {
		return "", ErrInvalidSeverityReason
	}
	return reason, nil
}

//対応メモの文字数を確かめる
func WorkNoteText(body string) (string, error) {
	body = strings.TrimSpace(body)
	if utf8.RuneCountInString(body) < 1 || utf8.RuneCountInString(body) > 5000 {
		return "", ErrInvalidWorkNote
	}
	return body, nil
}

//対応メモを書いてよいかを確かめる
func CanAddWorkNote(role Role, actorID string, ticket Ticket) error {
	if role == RoleAdmin {
		if ticket.Status == TicketClosed {
			return ErrWorkNoteClosed
		}
		return nil
	}
	if role == RoleAgent {
		if ticket.Status != TicketInProgress || ticket.AssigneeID != actorID {
			return ErrWorkNoteAgent
		}
		return nil
	}
	return ErrWorkNoteRole
}

//チケットの詳細を見られるか確かめる
func CanViewTicket(role Role, actorID string, ticket Ticket) error {
	switch role {
	case RoleAdmin:
		return nil
	case RoleApplicant:
		if ticket.CreatedBy != actorID {
			return ErrTicketNotFound
		}
		return nil
	case RoleAgent:
		if ticket.Status == TicketOpen || ticket.AssigneeID == actorID {
			return nil
		}
		return ErrTicketHidden
	default:
		return ErrTicketNotFound
	}
}

//緊急度を変えてよいかを確かめる
func CanChangeSeverity(role Role, actorID string, ticket Ticket, next int) error {
	if role != RoleAgent && role != RoleAdmin {
		return ErrSeverityRole
	}
	if next < 1 || next > 4 {
		return ErrInvalidSeverity
	}
	if ticket.Status == TicketClosed {
		return ErrSeverityClosed
	}
	if next == ticket.Severity {
		return ErrSeveritySame
	}
	if next < ticket.Severity && role != RoleAdmin {
		return ErrSeverityLower
	}
	if role == RoleAgent && (ticket.Status != TicketInProgress || ticket.AssigneeID != actorID) {
		return ErrSeverityRaise
	}
	return nil
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
