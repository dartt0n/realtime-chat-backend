package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/dartt0n/realtime-chat-backend/forms"
	"github.com/dartt0n/realtime-chat-backend/kv"
	"github.com/dartt0n/realtime-chat-backend/models"
	"github.com/google/uuid"
	"github.com/tinode/chat/pbx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type TinodeService struct {
	kv     kv.KeyValueStore
	client pbx.NodeClient
	stream pbx.Node_MessageLoopClient

	auth *AuthService

	reqres *sync.Map // chan any
}

func NewTinodeService(addr string, kv kv.KeyValueStore, auth *AuthService) (*TinodeService, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client := pbx.NewNodeClient(conn)
	stream, err := client.MessageLoop(context.Background())
	if err != nil {
		return nil, err
	}

	t := &TinodeService{
		kv:     kv,
		client: client,
		stream: stream,
		auth:   auth,

		reqres: &sync.Map{},
	}

	go t.ListenUpdates()

	rID := uuid.NewString()
	if _, err := t.Send(rID, &pbx.ClientMsg{Message: &pbx.ClientMsg_Hi{
		Hi: &pbx.ClientHi{
			Id:        rID,
			UserAgent: "golang/1.0",
			Ver:       "0.22.13",
			Lang:      "EN",
		},
	}}); err != nil {
		return nil, err
	}

	return t, nil
}

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
		case *pbx.ServerMsg_Info:
			slog.Info("received info message", "topic", m.Info.Topic, "msg", m.Info.What.String())
		default:
			slog.Error("received unknown message", "message", msg)
		}
	}
}

func (s TinodeService) declareReq(rID string) error {
	if _, ok := s.reqres.Load(rID); ok {
		return errors.New("dublicate request id")
	}
	s.reqres.Store(rID, make(chan any, 1))
	slog.Debug("declared request", "id", rID)

	return nil
}

func (s TinodeService) revokeReq(rID string) error {
	ch, ok := s.reqres.LoadAndDelete(rID)
	if !ok {
		return errors.New("request not found")
	}
	close(ch.(chan any))
	slog.Debug("revoked request", "id", rID)
	return nil
}

func (s TinodeService) Send(rID string, msg *pbx.ClientMsg) (res any, err error) {
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

	// await for res from event loop
	res = <-ch.(chan any)
	slog.Debug("received response", "res", res)

	return res, nil
}

func generateUsername(email string) string {
	email = strings.ToLower(strings.Trim(email, " \n\r\t"))

	prefix := strings.Split(email, "@")[0]
	provider := strings.Split(strings.Split(email, "@")[1], ".")[0][:2]

	emailhash := md5.Sum([]byte(email))
	shorthash := hex.EncodeToString(emailhash[:])[:8]
	return prefix + "_" + provider + "_" + shorthash
}

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

	// await for res from event loop
	rawres, err := s.Send(rID, req)
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

	user.ID = models.UserID(res.Ctrl.Params["user"])
	user.Email = form.Email
	user.Password = form.Password

	return user, nil
}

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

	// await for res from event loop
	rawres, err := s.Send(rID, req)
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

	td, err := s.auth.CreateToken(models.UserID(res.Ctrl.Params["user_id"]))
	if err != nil {
		slog.Error("failed to create token", "error", err)
		return user, token, err
	}

	err = s.auth.CreateAuth(models.UserID(res.Ctrl.Params["user_id"]), td)
	if err != nil {
		slog.Error("failed to create auth", "error", err)
		return user, token, err
	}

	token.AccessToken = td.AccessToken
	token.RefreshToken = td.RefreshToken

	s.kv.Set(td.AccessUUID+":token", string(res.Ctrl.Params["token"]), 10*time.Minute)

	user.ID = models.UserID(res.Ctrl.Params["user_id"])
	user.Email = form.Email
	user.Password = form.Password

	return user, token, nil
}
