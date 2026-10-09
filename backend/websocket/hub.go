package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"

	"escalator/entity"
	"escalator/usecase"
)

type client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	userID string
	role   entity.Role
}

type Hub struct {
	clients    map[*client]struct{}
	register   chan *client
	unregister chan *client
	broadcast  chan usecase.Message
	current    *usecase.CurrentUser
	snapshot   *usecase.QueueSnapshot
	dashboard  *usecase.ShowDashboard
	upgrader   websocket.Upgrader
	mu         sync.Mutex
	goneSince  map[string]time.Time
}

func NewHub(current *usecase.CurrentUser, snapshot *usecase.QueueSnapshot, dashboard *usecase.ShowDashboard, origin string) *Hub {
	hub := &Hub{
		clients:    make(map[*client]struct{}),
		register:   make(chan *client),
		unregister: make(chan *client),
		broadcast:  make(chan usecase.Message, 32),
		current:    current,
		snapshot:   snapshot,
		dashboard:  dashboard,
		goneSince:  make(map[string]time.Time),
	}
	hub.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			got := r.Header.Get("Origin")
			if got == "" {
				return true
			}
			return got == origin
		},
	}
	return hub
}

//つながっている画面へ知らせを配る
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = struct{}{}
			if client.role == entity.RoleAgent {
				h.ForgetGone(client.userID)
			}
		case client := <-h.unregister:
			h.removeClient(client)
		case message := <-h.broadcast:
			for client := range h.clients {
				if !client.wants(message) {
					continue
				}
				select {
				case client.send <- message.Body:
				default:
					h.removeClient(client)
					log.Printf("画面への知らせを届けられなかったため、接続を切りました。user_id=%s", client.userID)
				}
			}
		}
	}
}

//保存が済んだあとの知らせを、つながっている画面へ届ける
func (h *Hub) Send(message usecase.Message) {
	select {
	case h.broadcast <- message:
	default:
		log.Printf("画面への知らせを届けられませんでした")
	}
}

func (h *Hub) removeClient(client *client) {
	if _, ok := h.clients[client]; !ok {
		return
	}
	delete(h.clients, client)
	close(client.send)
	if client.role != entity.RoleAgent || h.agentConnected(client.userID) {
		return
	}
	h.markGone(client.userID)
}

func (h *Hub) agentConnected(userID string) bool {
	for client := range h.clients {
		if client.role == entity.RoleAgent && client.userID == userID {
			return true
		}
	}
	return false
}

func (h *Hub) markGone(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.goneSince[userID]; ok {
		return
	}
	h.goneSince[userID] = time.Now()
}

//接続が切れたまま、指定時刻まで戻っていない担当者を返す
func (h *Hub) AgentsGoneSince(before time.Time) []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	ids := make([]string, 0)
	for id, at := range h.goneSince {
		if !at.After(before) {
			ids = append(ids, id)
		}
	}
	return ids
}

//担当者の接続が、まだすべて切れたままかを返す
func (h *Hub) StillGone(userID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, ok := h.goneSince[userID]
	return ok
}

//担当者が戻った、または離席にしたあと、切れた記録を消す
func (h *Hub) ForgetGone(userID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.goneSince, userID)
}

func (c *client) wants(message usecase.Message) bool {
	if message.Admins && c.role == entity.RoleAdmin {
		return true
	}
	if message.Staff && (c.role == entity.RoleAgent || c.role == entity.RoleAdmin) {
		return true
	}
	return message.ApplicantID != "" && c.role == entity.RoleApplicant && c.userID == message.ApplicantID
}

//ログインを確かめて接続し、担当者と管理者には今の待ち順と稼働を送る
func (h *Hub) Serve(c echo.Context) error {
	token := accessToken(c)
	if token == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"message": entity.ErrUnauthenticated.Error()})
	}
	actor, err := h.current.Execute(c.Request().Context(), "Bearer "+token)
	if err != nil {
		if errors.Is(err, entity.ErrUnauthenticated) {
			return c.JSON(http.StatusUnauthorized, map[string]string{"message": err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "ログインを確認できませんでした。しばらくしてから、もう一度試してください"})
	}
	conn, err := h.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	person := &client{
		hub:    h,
		conn:   conn,
		send:   make(chan []byte, 16),
		userID: actor.ID,
		role:   actor.Role,
	}
	if actor.Role == entity.RoleAgent || actor.Role == entity.RoleAdmin {
		if err := h.writeSnapshot(c.Request().Context(), conn); err != nil {
			log.Printf("画面への最初の待ち順を送れませんでした: %v", err)
			conn.Close()
			return nil
		}
	}
	if actor.Role == entity.RoleAdmin {
		if err := h.writeDashboard(c.Request().Context(), conn); err != nil {
			log.Printf("画面への最初の現場の数字を送れませんでした: %v", err)
			conn.Close()
			return nil
		}
	}
	h.register <- person
	go person.write()
	person.read()
	return nil
}

func accessToken(c echo.Context) string {
	if token := strings.TrimSpace(c.QueryParam("token")); token != "" {
		return token
	}
	const prefix = "Bearer "
	header := c.Request().Header.Get("Authorization")
	if strings.HasPrefix(header, prefix) {
		return strings.TrimSpace(header[len(prefix):])
	}
	return ""
}

func (h *Hub) writeSnapshot(ctx context.Context, conn *websocket.Conn) error {
	picture, err := h.snapshot.Execute(ctx)
	if err != nil {
		log.Printf("画面への最初の待ち順を用意できませんでした: %v", err)
		return nil
	}
	tickets := make([]snapshotTicket, 0, len(picture.Tickets))
	for _, ticket := range picture.Tickets {
		tickets = append(tickets, snapshotTicket{
			Rank:             ticket.Rank,
			ID:               ticket.ID,
			Title:            ticket.Title,
			Severity:         ticket.Severity,
			Plan:             string(ticket.Plan),
			WaitMinutes:      ticket.WaitMinutes,
			PriorityScore:    ticket.PriorityScore,
			RemainingMinutes: ticket.RemainingMinutes,
			Overdue:          ticket.Overdue,
			OverdueMinutes:   ticket.OverdueMinutes,
			CustomerName:     ticket.CustomerName,
		})
	}
	agents := make([]snapshotAgent, 0, len(picture.Agents))
	for _, agent := range picture.Agents {
		agents = append(agents, snapshotAgent{UserID: agent.UserID, Name: agent.Name, Status: agent.Status})
	}
	body, err := json.Marshal(snapshotEvent{Type: "queue_snapshot", Tickets: tickets, Agents: agents})
	if err != nil {
		return err
	}
	if err := conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, body)
}

func (h *Hub) writeDashboard(ctx context.Context, conn *websocket.Conn) error {
	numbers, err := h.dashboard.Collect(ctx)
	if err != nil {
		log.Printf("画面への最初の現場の数字を用意できませんでした: %v", err)
		return nil
	}
	body, err := json.Marshal(usecase.DashboardMessage(numbers))
	if err != nil {
		return err
	}
	if err := conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, body)
}

func (c *client) read() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(512)
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (c *client) write() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case body, ok := <-c.send:
			if !ok {
				return
			}
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, body); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
