package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/dartt0n/realtime-chat-backend/forms"
	"github.com/dartt0n/realtime-chat-backend/kv"
	"github.com/dartt0n/realtime-chat-backend/models"
	"github.com/google/uuid"
	"github.com/tinode/chat/pbx"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// TinodeService handles communication with the Tinode chat server
// It manages user authentication, message passing and server updates
type TinodeService struct {
	kv     kv.KeyValueStore           // Key-value store for persistent data
	client pbx.NodeClient             // gRPC client for Tinode server
	stream pbx.Node_MessageLoopClient // Bi-directional message stream

	auth   *AuthService
	topic  models.Topic
	reqres *sync.Map // Maps request IDs to response channels

	mongouri string
	mongodb  string
}

// NewTinodeService creates a new TinodeService instance
// addr: Tinode server address (e.g. "localhost:6061")
// kv: Key-value store implementation
// auth: Authentication service instance
func NewTinodeService(addr string, generalTopic models.Topic, mongouri, mongodb string, kv kv.KeyValueStore, auth *AuthService) (*TinodeService, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client := pbx.NewNodeClient(conn)
	stream, err := client.MessageLoop(context.Background())
	if err != nil {
		return nil, err
	}

	s := &TinodeService{
		kv:       kv,
		client:   client,
		stream:   stream,
		auth:     auth,
		topic:    generalTopic,
		reqres:   &sync.Map{},
		mongouri: mongouri,
		mongodb:  mongodb,
	}

	go s.ListenUpdates()

	// Send initial handshake message
	if err := s.ping(); err != nil {
		return nil, err
	}

	return s, nil
}

// ListenUpdates handles incoming messages from the Tinode server
// It processes different types of messages (control, data, presence etc.)
// and routes responses to the appropriate request handlers
func (s TinodeService) ListenUpdates() {
	for {
		msg, err := s.stream.Recv()
		if err != nil {
			slog.Error("failed to receive message", "error", err)
			return
		}

		switch m := msg.Message.(type) {
		case *pbx.ServerMsg_Ctrl:
			slog.Info("received control message", "code", m.Ctrl.Code, "msg", m.Ctrl.Text)

			// Route control message to waiting request handler if one exists
			if ch, ok := s.reqres.Load(m.Ctrl.Id); ok {
				ch.(chan any) <- m
			} else {
				slog.Warn("received unawaited control message", "code", m.Ctrl.Code, "msg", m.Ctrl.Text, "id", m.Ctrl.Id)
			}
		case *pbx.ServerMsg_Data:
			slog.Info("received data message", "topic", m.Data.Topic, "msg", "<bytes>", "length", len(m.Data.Content))
		case *pbx.ServerMsg_Pres:
			slog.Info("received presence message", "topic", m.Pres.Topic, "msg", m.Pres.What.String())
		case *pbx.ServerMsg_Meta:
			slog.Info("received metadata message", "topic", m.Meta.Topic, "msg", m.Meta.Desc)

			// Route meta message to waiting request handler if one exists
			if ch, ok := s.reqres.Load(m.Meta.Id); ok {
				ch.(chan any) <- m
			} else {
				slog.Warn("received unawaited meta message", "topic", m.Meta.Topic, "msg", m.Meta.Desc)
			}
		case *pbx.ServerMsg_Info:
			slog.Info("received info message", "topic", m.Info.Topic, "msg", m.Info.What.String())
		default:
			slog.Error("received unknown message", "message", msg)
		}
	}
}

// ping sends a Hi message to the server and waits for a response
func (s TinodeService) ping() error {
	rID := uuid.NewString()
	_, err := s.send(rID, &pbx.ClientMsg{Message: &pbx.ClientMsg_Hi{
		Hi: &pbx.ClientHi{
			Id:        rID,
			UserAgent: "golang/1.0",
			Ver:       "0.22.13",
			Lang:      "EN",
		},
	}})

	return err
}

// CreateUser registers a new user with the Tinode server
// form: Registration form containing email and password
// Returns the created user model and any error
func (s TinodeService) CreateUser(form forms.RegisterForm) (user models.User, err error) {
	rID := uuid.NewString()
	username := generateUsername(form.Email)

	publicPayload, err := json.Marshal(map[string]any{
		"username": username,
	})
	if err != nil {
		return user, err
	}

	privatePayload, err := json.Marshal(map[string]any{
		"email": form.Email,
	})
	if err != nil {
		return user, err
	}

	req := &pbx.ClientMsg{Message: &pbx.ClientMsg_Acc{
		Acc: &pbx.ClientAcc{
			Id:     rID,
			UserId: "new" + username,
			Scheme: "basic",
			Secret: []byte(username + ":" + form.Password),
			Login:  false,
			Tags:   []string{},
			Desc: &pbx.SetDesc{
				DefaultAcs: &pbx.DefaultAcsMode{
					Auth: "JRWPA",
					Anon: "N",
				},
				Public:  publicPayload,
				Private: privatePayload,
			},
		},
	}}
	slog.Debug("sending account registration message", "id", rID, "msg", req)

	rawres, err := s.send(rID, req)
	if err != nil {
		slog.Error("failed to send account registration message", "error", err, "id", rID)
		return user, err
	}

	res, ok := rawres.(*pbx.ServerMsg_Ctrl)
	if !ok {
		slog.Error("failed to project type to ServerMsg_Ctrl", "id", rID, "res", rawres)
		return user, errors.New("unexpected response from event loop")
	}
	slog.Debug("received response from event loop", "res", res)

	if res.Ctrl.Code != 201 {
		return user, errors.New("unexpected response code")
	}

	user.ID = models.UserID(strings.Trim(string(res.Ctrl.Params["user"]), "\""))
	user.Email = form.Email
	user.Password = form.Password

	return user, nil
}

// Login authenticates a user with the Tinode server
// form: Login form containing email and password
// Returns the user model, authentication tokens and any error
func (s TinodeService) Login(form forms.LoginForm) (user models.User, token models.Token, err error) {
	rID := uuid.NewString()
	username := generateUsername(form.Email)

	req := &pbx.ClientMsg{Message: &pbx.ClientMsg_Login{
		Login: &pbx.ClientLogin{
			Id:     rID,
			Scheme: "basic",
			Secret: []byte(username + ":" + form.Password),
		},
	}}

	rawres, err := s.send(rID, req)
	if err != nil {
		slog.Error("failed to send login message", "error", err, "id", rID)
		return user, token, err
	}

	res, ok := rawres.(*pbx.ServerMsg_Ctrl)
	if !ok {
		slog.Error("failed to project type to ServerMsg_Ctrl", "id", rID, "res", rawres)
		return user, token, errors.New("unexpected response from event loop")
	}
	slog.Debug("received response from event loop", "res", res)

	if res.Ctrl.Code != 200 {
		slog.Error("unexpected response code", "code", res.Ctrl.Code, "res", res)
		return user, token, errors.New("unexpected response code")
	}

	td, err := s.auth.CreateToken(models.UserID(strings.Trim(string(res.Ctrl.Params["user"]), "\"")))
	if err != nil {
		slog.Error("failed to create token", "error", err)
		return user, token, err
	}

	err = s.auth.CreateAuth(models.UserID(strings.Trim(string(res.Ctrl.Params["user"]), "\"")), td)
	if err != nil {
		slog.Error("failed to create auth", "error", err)
		return user, token, err
	}

	token.AccessToken = td.AccessToken
	token.RefreshToken = td.RefreshToken

	s.kv.Set(td.AccessUUID+":token", string(res.Ctrl.Params["token"]), 0)

	if err := s.joinTopic(s.topic.ID); err != nil {
		slog.Error("failed to join topic", "error", err)
		return user, token, err
	}

	user.ID = models.UserID(strings.Trim(string(res.Ctrl.Params["user"]), "\""))
	user.Email = form.Email
	user.Password = form.Password
	return user, token, nil
}

func (s TinodeService) FetchLastMsgs() ([]models.Message, error) {
	conn, err := mongo.Connect(options.Client().ApplyURI(s.mongouri))
	if err != nil {
		return nil, err
	}
	defer conn.Disconnect(context.Background())

	db := conn.Database(s.mongodb)

	cursor, err := db.Collection("messages").Find(context.Background(), bson.D{bson.E{Key: "topic", Value: s.topic.ID}}, options.Find().SetSort(bson.D{bson.E{Key: "seq_id", Value: -1}}).SetLimit(50))
	if err != nil {
		slog.Error("failed to fetch last messages", "error", err)
		return nil, err
	}
	defer cursor.Close(context.Background())

	var messages []models.Message
	if err := cursor.All(context.Background(), &messages); err != nil {
		slog.Error("failed to fetch all messages", "error", err)
		return nil, err
	}

	return messages, nil
}

func (s TinodeService) SendMessage(accessUUID, content string) error {
	rID := uuid.NewString()

	// if err := s.switchUser(accessUUID); err != nil {
	// 	return err
	// }

	msg := &pbx.ClientMsg{
		Message: &pbx.ClientMsg_Pub{
			Pub: &pbx.ClientPub{
				Id:      rID,
				Topic:   s.topic.ID,
				Content: []byte(fmt.Sprintf(`"%s"`, content)),
				NoEcho:  false,
			},
		},
	}

	res, err := s.send(rID, msg)
	if err != nil {
		return err
	}

	if res.(*pbx.ServerMsg_Ctrl).Ctrl.Code/100 != 2 {
		return errors.New("unexpected response code")
	}

	return nil
}

// declareReq creates a new response channel for a request ID
func (s TinodeService) declareReq(rID string) error {
	if _, ok := s.reqres.Load(rID); ok {
		return errors.New("dublicate request id")
	}
	// unfortunately, go type system doesn't allow us to create discriminated unions, so we have to use any (interface{})
	s.reqres.Store(rID, make(chan any, 1))
	slog.Debug("declared request", "id", rID)

	return nil
}

// revokeReq removes a response channel for a request ID
func (s TinodeService) revokeReq(rID string) error {
	ch, ok := s.reqres.LoadAndDelete(rID)
	if !ok {
		return errors.New("request not found")
	}
	close(ch.(chan any))
	slog.Debug("revoked request", "id", rID)
	return nil
}

// send transmits a message to the Tinode server and waits for a response
// rID: Request ID for tracking the response
// msg: Message to send
// Returns the server response and any error
func (s TinodeService) send(rID string, msg *pbx.ClientMsg) (res any, err error) {
	err = s.declareReq(rID)
	if err != nil {
		slog.Error("failed to declare request", "error", err, "id", rID)
		return res, err
	}

	defer func() {
		err := s.revokeReq(rID)
		if err != nil {
			slog.Error("failed to revoke request", "error", err, "id", rID)
		}
	}()

	slog.Debug("sending message", "id", rID, "msg", msg)
	err = s.stream.Send(msg)
	if err != nil {
		slog.Error("failed to send account registration message", "error", err, "id", rID)
		return res, err
	}
	slog.Debug("awaiting for response", "id", rID)
	ch, ok := s.reqres.Load(rID)
	if !ok {
		return res, errors.New("internal error")
	}

	// Wait for response from event loop
	res = <-ch.(chan any)
	slog.Debug("received response", "res", res)

	return res, nil
}

// generateUsername creates a unique username from an email address
// Format: localpart_pr_hash where:
// - localpart is the part before @ in email
// - pr is first 2 chars of provider name
// - hash is first 8 chars of MD5 hash of full email
// Example: john_gm_5d41402a for john@gmail.com
func generateUsername(email string) string {
	email = strings.ToLower(strings.Trim(email, " \n\r\t"))

	prefix := strings.Split(email, "@")[0]
	provider := strings.Split(strings.Split(email, "@")[1], ".")[0][:2]

	emailhash := md5.Sum([]byte(email))
	shorthash := hex.EncodeToString(emailhash[:])[:8]
	return prefix + "_" + provider + "_" + shorthash
}

func (s TinodeService) userToken(user, pass string) (string, error) {
	rID := uuid.NewString()

	req := &pbx.ClientMsg{Message: &pbx.ClientMsg_Login{
		Login: &pbx.ClientLogin{
			Id:     rID,
			Scheme: "basic",
			Secret: []byte(user + ":" + pass),
		},
	}}

	rawres, err := s.send(rID, req)
	if err != nil {
		slog.Error("failed to send login message", "error", err, "id", rID)
		return "", err
	}

	res, ok := rawres.(*pbx.ServerMsg_Ctrl)
	if !ok {
		slog.Error("failed to project type to ServerMsg_Ctrl", "id", rID, "res", rawres)
		return "", errors.New("unexpected response from event loop")
	}
	slog.Debug("received response from event loop", "res", res)

	if res.Ctrl.Code != 200 {
		slog.Error("unexpected response code", "code", res.Ctrl.Code, "res", res)
		return "", errors.New("unexpected response code")
	}

	return string(res.Ctrl.Params["token"]), nil
}

func (s TinodeService) joinTopic(topicID string) (err error) {
	rID := uuid.NewString()

	msg := &pbx.ClientMsg{Message: &pbx.ClientMsg_Sub{
		Sub: &pbx.ClientSub{
			Id:    rID,
			Topic: topicID,
			SetQuery: &pbx.SetQuery{
				Desc: &pbx.SetDesc{
					DefaultAcs: &pbx.DefaultAcsMode{
						Auth: "JRWPA",
						Anon: "N",
					},
				},
			},
		},
	}}

	rawres, err := s.send(rID, msg)
	if err != nil {
		slog.Error("failed to send topic creation message", "error", err, "id", rID)
		return err
	}

	res, ok := rawres.(*pbx.ServerMsg_Ctrl)
	if !ok {
		slog.Error("failed to project type to ServerMsg_Ctrl", "id", rID, "res", rawres)
		return errors.New("unexpected response from event loop")
	}
	slog.Debug("received response from event loop", "res", res)

	if res.Ctrl.Code != 200 {
		return errors.New("unexpected response code")
	}

	return nil
}

func (s TinodeService) switchUser(accessUUID string) (err error) {
	rID := uuid.NewString()
	token, err := s.kv.Get(accessUUID + ":token")
	if err != nil {
		slog.Error("failed to get token", "error", err)
		return err
	}

	req := &pbx.ClientMsg{Message: &pbx.ClientMsg_Login{
		Login: &pbx.ClientLogin{
			Id:     rID,
			Scheme: "token",
			Secret: []byte(token),
		},
	}}

	rawres, err := s.send(rID, req)
	if err != nil {
		slog.Error("failed to send login message", "error", err, "id", rID)
		return err
	}

	res, ok := rawres.(*pbx.ServerMsg_Ctrl)
	if !ok {
		slog.Error("failed to project type to ServerMsg_Ctrl", "id", rID, "res", rawres)
		return errors.New("unexpected response from event loop")
	}
	slog.Debug("received response from event loop", "res", res)

	if res.Ctrl.Code != 200 {
		slog.Error("unexpected response code", "code", res.Ctrl.Code, "res", res)
		return errors.New("unexpected response code")
	}

	return nil
}

func (s TinodeService) getLastMsgID() (int32, error) {
	rID := uuid.NewString()

	msg := &pbx.ClientMsg{Message: &pbx.ClientMsg_Get{
		Get: &pbx.ClientGet{
			Id:    rID,
			Topic: s.topic.ID,
			Query: &pbx.GetQuery{
				What: "desc",
			},
		},
	}}

	rawres, err := s.send(rID, msg)
	if err != nil {
		slog.Error("failed to send message", "error", err, "id", rID)
		return 0, err
	}

	res, ok := rawres.(*pbx.ServerMsg_Meta)
	if !ok {
		slog.Error("failed to project type to ServerMsg_Ctrl", "id", rID, "res", rawres)
		return 0, errors.New("unexpected response from event loop")
	}
	slog.Debug("received response from event loop", "res", res)

	return res.Meta.Desc.SeqId, nil
}
