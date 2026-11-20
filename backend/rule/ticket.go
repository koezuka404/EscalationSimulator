package rule

import (
	"strings"
	"time"
	"unicode/utf8"

	"escalator/entity"
)

//状態が対応待ち、対応中、完了のどれか確かめる
func ParseTicketStatus(raw string) (entity.TicketStatus, error) {
	switch entity.TicketStatus(raw) {
	case entity.TicketOpen, entity.TicketInProgress, entity.TicketClosed:
		return entity.TicketStatus(raw), nil
	default:
		return "", ErrInvalidTicketStatus
	}
}

//件名、詳細、種類、緊急度を確かめて、対応待ちのチケットを作る
func NewTicket(customerID, createdBy, title, description string, severity int, category entity.Category, createdAt time.Time) (entity.Ticket, error) {
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	if customerID == "" {
		return entity.Ticket{}, ErrCustomerRequired
	}
	if utf8.RuneCountInString(title) < 1 || utf8.RuneCountInString(title) > 200 {
		return entity.Ticket{}, ErrInvalidTitle
	}
	if utf8.RuneCountInString(description) < 1 || utf8.RuneCountInString(description) > 5000 {
		return entity.Ticket{}, ErrInvalidDescription
	}
	if severity < 1 || severity > 4 {
		return entity.Ticket{}, ErrInvalidSeverity
	}
	if !ValidCategory(category) {
		return entity.Ticket{}, ErrInvalidCategory
	}
	return entity.Ticket{
		CustomerID:  customerID,
		CreatedBy:   createdBy,
		Title:       title,
		Description: description,
		Severity:    severity,
		Category:    category,
		Status:      entity.TicketOpen,
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
func CanAddWorkNote(role entity.Role, actorID string, ticket entity.Ticket) error {
	if role == entity.RoleAdmin {
		if ticket.Status == entity.TicketClosed {
			return ErrWorkNoteClosed
		}
		return nil
	}
	if role == entity.RoleAgent {
		if ticket.Status != entity.TicketInProgress || ticket.AssigneeID != actorID {
			return ErrWorkNoteAgent
		}
		return nil
	}
	return ErrWorkNoteRole
}

//チケットの詳細を見られるか確かめる
func CanViewTicket(role entity.Role, actorID string, ticket entity.Ticket) error {
	switch role {
	case entity.RoleAdmin:
		return nil
	case entity.RoleApplicant:
		if ticket.CreatedBy != actorID {
			return ErrTicketNotFound
		}
		return nil
	case entity.RoleAgent:
		if ticket.Status == entity.TicketOpen || ticket.AssigneeID == actorID {
			return nil
		}
		return ErrTicketHidden
	default:
		return ErrTicketNotFound
	}
}

//緊急度を変えてよいかを確かめる
func CanChangeSeverity(role entity.Role, actorID string, ticket entity.Ticket, next int) error {
	if role != entity.RoleAgent && role != entity.RoleAdmin {
		return ErrSeverityRole
	}
	if next < 1 || next > 4 {
		return ErrInvalidSeverity
	}
	if ticket.Status == entity.TicketClosed {
		return ErrSeverityClosed
	}
	if next == ticket.Severity {
		return ErrSeveritySame
	}
	if next < ticket.Severity && role != entity.RoleAdmin {
		return ErrSeverityLower
	}
	if role == entity.RoleAgent && (ticket.Status != entity.TicketInProgress || ticket.AssigneeID != actorID) {
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

func ValidCategory(category entity.Category) bool {
	switch category {
	case entity.CategoryIncident, entity.CategoryBug, entity.CategoryQuestion, entity.CategoryRequest, entity.CategoryOther:
		return true
	default:
		return false
	}
}
