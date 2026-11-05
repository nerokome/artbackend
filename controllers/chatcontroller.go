package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/nerokome/artfolio-backend/database"
	"github.com/nerokome/artfolio-backend/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type chatMessage struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ConversationID string             `bson:"conversationId" json:"conversationId"`
	SenderID       primitive.ObjectID `bson:"senderId" json:"senderId"`
	RecipientID    primitive.ObjectID `bson:"recipientId" json:"recipientId"`
	Text           string             `bson:"text" json:"text"`
	CreatedAt      time.Time          `bson:"createdAt" json:"createdAt"`
}

func conversationKey(a, b primitive.ObjectID) string {
	ids := []string{a.Hex(), b.Hex()}
	sort.Strings(ids)
	return ids[0] + ":" + ids[1]
}

func GetChatHistory(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}
	artistID, err := primitive.ObjectIDFromHex(c.Param("artistId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid artist id"})
		return
	}
	if artistID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot start a chat with yourself"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	if err := database.Collection("users").FindOne(ctx, bson.M{"_id": artistID}).Err(); err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "artist not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find artist"})
		}
		return
	}
	limit := int64(50)
	if raw := c.Query("limit"); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	conversationID := conversationKey(userID, artistID)
	cur, err := database.Collection("messages").Find(ctx, bson.M{"conversationId": conversationID}, options.Find().SetSort(bson.M{"createdAt": -1}).SetLimit(limit))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch messages"})
		return
	}
	defer cur.Close(ctx)
	var messages []chatMessage
	if err := cur.All(ctx, &messages); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse messages"})
		return
	}
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	if messages == nil {
		messages = []chatMessage{}
	}
	c.JSON(http.StatusOK, gin.H{"conversationId": conversationID, "messages": messages})
}

type chatPeer struct {
	conn *websocket.Conn
	send chan []byte
}
type websocketHub struct {
	mu    sync.Mutex
	rooms map[string]map[*chatPeer]struct{}
}

var chatConnections = websocketHub{rooms: make(map[string]map[*chatPeer]struct{})}

func (h *websocketHub) join(room string, peer *chatPeer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[room] == nil {
		h.rooms[room] = make(map[*chatPeer]struct{})
	}
	h.rooms[room][peer] = struct{}{}
}
func (h *websocketHub) leave(room string, peer *chatPeer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.rooms[room], peer)
	if len(h.rooms[room]) == 0 {
		delete(h.rooms, room)
	}
}
func (h *websocketHub) publish(room string, payload []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for peer := range h.rooms[room] {
		select {
		case peer.send <- payload:
		default:
			_ = peer.conn.Close()
			delete(h.rooms[room], peer)
		}
	}
}

var chatUpgrader = websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024, CheckOrigin: allowedChatOrigin}

func allowedChatOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	allowed := map[string]bool{"localhost:3000": true, "127.0.0.1:3000": true, "artwork-phi-swart.vercel.app": true}
	for _, item := range strings.Split(os.Getenv("CORS_ALLOWED_ORIGINS"), ",") {
		if parsed, e := url.Parse(strings.TrimSpace(item)); e == nil && parsed.Host != "" {
			allowed[parsed.Host] = true
		}
	}
	return allowed[strings.ToLower(u.Host)]
}

func ArtistChatWebSocket(c *gin.Context) {
	token := c.Query("token")
	claims, err := utils.ValidateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "valid token query parameter required"})
		return
	}
	userIDStr, _ := claims["user_id"].(string)
	if userIDStr == "" {
		userIDStr, _ = claims["userId"].(string)
	}
	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user id in token"})
		return
	}
	artistID, err := primitive.ObjectIDFromHex(c.Param("artistId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid artist id"})
		return
	}
	if artistID == userID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot chat with yourself"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	err = database.Collection("users").FindOne(ctx, bson.M{"_id": artistID}).Err()
	cancel()
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "artist not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to find artist"})
		}
		return
	}
	conn, err := chatUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	room := conversationKey(userID, artistID)
	peer := &chatPeer{conn: conn, send: make(chan []byte, 32)}
	chatConnections.join(room, peer)
	defer func() { chatConnections.leave(room, peer); conn.Close() }()
	go peer.writeLoop()
	conn.SetReadLimit(6000)
	conn.SetReadDeadline(time.Now().Add(70 * time.Second))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(70 * time.Second)) })
	for {
		_, payload, err := conn.ReadMessage()
		if err != nil {
			break
		}
		var incoming struct {
			Text string `json:"text"`
		}
		if json.Unmarshal(payload, &incoming) != nil {
			continue
		}
		text := strings.TrimSpace(incoming.Text)
		if text == "" || len([]rune(text)) > 5000 {
			continue
		}
		message := chatMessage{ID: primitive.NewObjectID(), ConversationID: room, SenderID: userID, RecipientID: artistID, Text: text, CreatedAt: time.Now().UTC()}
		writeCtx, writeCancel := context.WithTimeout(context.Background(), 10*time.Second)
		_, err = database.Collection("messages").InsertOne(writeCtx, message)
		if err == nil {
			var sender struct {
				FullName string `bson:"full_name"`
			}
			_ = database.Collection("users").FindOne(writeCtx, bson.M{"_id": userID}).Decode(&sender)
			preview := text
			if len([]rune(preview)) > 120 {
				preview = string([]rune(preview)[:120]) + "…"
			}
			createNotification(writeCtx, artistID, userID, "message", room, sender.FullName+" sent you a message: "+preview)
		}
		writeCancel()
		if err != nil {
			continue
		}
		encoded, err := json.Marshal(message)
		if err == nil {
			chatConnections.publish(room, encoded)
		}
	}
}

func (p *chatPeer) writeLoop() {
	ticker := time.NewTicker(50 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case payload := <-p.send:
			p.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := p.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return
			}
		case <-ticker.C:
			p.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := p.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
