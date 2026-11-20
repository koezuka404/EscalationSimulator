package controller

import (
	"context"

	"escalator/entity"
	"escalator/usecase"
)

type signUpUsecase interface {
	Execute(ctx context.Context, name, email, password string) (entity.User, error)
}

type logInUsecase interface {
	Execute(ctx context.Context, email, password string) (usecase.LoginResult, error)
}

type logOutUsecase interface {
	Execute(ctx context.Context, refreshToken string) error
}

type refreshUsecase interface {
	Execute(ctx context.Context, refreshToken string) (usecase.LoginResult, error)
}

type currentUserUsecase interface {
	Execute(ctx context.Context, authorization string) (entity.User, error)
}

type customersUsecase interface {
	List(ctx context.Context, authorization string) ([]entity.Customer, error)
	Create(ctx context.Context, authorization, name string, plan entity.Plan, slaMinutes int) (entity.Customer, error)
	Update(ctx context.Context, authorization, id, name string, plan entity.Plan, slaMinutes int) (entity.Customer, error)
}

type searchTicketsUsecase interface {
	Execute(ctx context.Context, authorization, status string, severity int, customerID, assigneeID string) ([]usecase.FoundTicket, error)
}

type createTicketUsecase interface {
	Execute(ctx context.Context, authorization, title, description, category string, severity int) (entity.Ticket, error)
}

type closeTicketUsecase interface {
	Execute(ctx context.Context, authorization, ticketID, comment string) (entity.Ticket, error)
}

type changeSeverityUsecase interface {
	Execute(ctx context.Context, authorization, ticketID, reason string, severity int) (entity.Ticket, error)
}

type returnTicketUsecase interface {
	Execute(ctx context.Context, authorization, ticketID string) (entity.Ticket, error)
}

type listMyTicketsUsecase interface {
	Execute(ctx context.Context, authorization string) ([]usecase.MyTicket, error)
}

type showTicketUsecase interface {
	Execute(ctx context.Context, authorization, ticketID string) (usecase.TicketDetail, error)
}

type addWorkNoteUsecase interface {
	Execute(ctx context.Context, authorization, ticketID, body string) (usecase.SavedWorkNote, error)
}

type listWaitingUsecase interface {
	Execute(ctx context.Context, authorization string) ([]usecase.WaitingTicket, error)
}

type claimTicketUsecase interface {
	Execute(ctx context.Context, authorization string) (entity.Ticket, error)
}

type changeAgentStatusUsecase interface {
	Execute(ctx context.Context, authorization, targetID, rawStatus string) (usecase.AgentStatus, error)
}

type showDashboardUsecase interface {
	Execute(ctx context.Context, authorization string) (usecase.Dashboard, error)
}

type linkApplicantUsecase interface {
	Execute(ctx context.Context, authorization, userID, customerID string) (entity.User, error)
}

type createAgentUsecase interface {
	Execute(ctx context.Context, authorization, name, email, password string) (entity.User, error)
}

type listUsersUsecase interface {
	Execute(ctx context.Context, authorization string) ([]entity.User, error)
}

type suspendUserUsecase interface {
	Execute(ctx context.Context, authorization, userID string) (entity.User, error)
}

type resumeUserUsecase interface {
	Execute(ctx context.Context, authorization, userID string) (entity.User, error)
}

type deleteUserUsecase interface {
	Execute(ctx context.Context, authorization, userID string) (entity.User, error)
}

type changeRoleUsecase interface {
	Execute(ctx context.Context, authorization, userID, role, customerID string) (entity.User, error)
}

type runDemoUsecase interface {
	Start(ctx context.Context, authorization string, count, intervalSec int, distribution string) (entity.DemoRun, error)
	Stop(ctx context.Context, authorization string) (entity.DemoRun, error)
}

type checkHealthUsecase interface {
	Execute(ctx context.Context) error
}
