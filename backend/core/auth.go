package core

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgxutil"
	"github.com/jackc/tpr/backend/data"
)

type Registration struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
type Credentials struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}
type Session struct {
	Name      string `json:"name"`
	SessionID string `json:"sessionID"`
}

func text(value string) pgtype.Text { return pgtype.Text{String: value, Valid: value != ""} }
func (c *Core) session(ctx context.Context, user *data.User) (Session, error) {
	id, err := genSessionID()
	if err != nil {
		return Session{}, err
	}
	err = data.InsertSession(ctx, c.pool, &data.Session{ID: id, UserID: user.ID.Int32})
	return Session{Name: user.Name.String, SessionID: hex.EncodeToString(id)}, err
}

var Register = Define(ActionDef[Registration, Session]{Name: "register", Description: "Register a reader and create a session", Metadata: ActionMetadata{Public: true, Mutating: true}, Handler: func(ctx context.Context, c *Core, p Registration) (Session, error) {
	if p.Name == "" {
		return Session{}, invalid(`Request must include the attribute "name"`)
	}
	if len(p.Name) > 30 {
		return Session{}, invalid(`"name" must be less than 30 characters`)
	}
	if err := validatePassword(p.Password); err != nil {
		return Session{}, invalid(err.Error())
	}
	u := &data.User{Name: text(p.Name), Email: text(p.Email)}
	if err := SetPassword(u, p.Password); err != nil {
		return Session{}, err
	}
	id, err := data.CreateUser(ctx, c.pool, u)
	if err != nil {
		var dup data.DuplicationError
		if errors.As(err, &dup) {
			return Session{}, invalid(fmt.Sprintf(`"%s" is already taken`, dup.Field))
		}
		return Session{}, err
	}
	u.ID = pgtype.Int4{Int32: id, Valid: true}
	return c.session(ctx, u)
}})
var CreateSession = Define(ActionDef[Credentials, Session]{Name: "create_session", Description: "Authenticate a reader and create a session", Metadata: ActionMetadata{Public: true, Mutating: true}, Handler: func(ctx context.Context, c *Core, p Credentials) (Session, error) {
	if p.Name == "" {
		return Session{}, invalid(`Request must include the attribute "name"`)
	}
	if p.Password == "" {
		return Session{}, invalid(`Request must include the attribute "password"`)
	}
	u, err := data.SelectUserByName(ctx, c.pool, p.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, invalid("Bad user name or password")
	}
	if err != nil {
		return Session{}, err
	}
	if !IsPassword(u, p.Password) {
		return Session{}, invalid("Bad user name or password")
	}
	return c.session(ctx, u)
}})

type SessionID struct {
	ID string `json:"id"`
}

var DeleteSession = Define(ActionDef[SessionID, Empty]{Name: "delete_session", Description: "Revoke a reader's session", Metadata: ActionMetadata{Mutating: true}, Handler: func(ctx context.Context, c *Core, p SessionID) (Empty, error) {
	id, err := hex.DecodeString(p.ID)
	if err != nil {
		return Empty{}, err
	}
	err = data.DeleteSessionByUserID(ctx, c.pool, id, User(ctx).ID.Int32)
	return Empty{}, err
}})

type Account struct {
	ID    int32  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

var GetAccount = Define(ActionDef[Empty, Account]{Name: "get_account", Description: "Read the current reader's account", Handler: func(ctx context.Context, c *Core, p Empty) (Account, error) {
	u := User(ctx)
	return Account{u.ID.Int32, u.Name.String, u.Email.String}, nil
}})

type AccountUpdate struct {
	Email            string `json:"email"`
	ExistingPassword string `json:"existingPassword"`
	NewPassword      string `json:"newPassword"`
}

var UpdateAccount = Define(ActionDef[AccountUpdate, Empty]{Name: "update_account", Description: "Update account email and password", Metadata: ActionMetadata{Mutating: true}, Handler: func(ctx context.Context, c *Core, p AccountUpdate) (Empty, error) {
	if !IsPassword(User(ctx), p.ExistingPassword) {
		return Empty{}, invalid("Bad existing password")
	}
	u, err := data.SelectUserByPK(ctx, c.pool, User(ctx).ID.Int32)
	if err != nil {
		return Empty{}, err
	}
	u.Email = text(p.Email)
	if p.NewPassword != "" {
		if err := SetPassword(u, p.NewPassword); err != nil {
			return Empty{}, err
		}
	}
	return Empty{}, data.UpdateUser(ctx, c.pool, u.ID.Int32, u)
}})

type PasswordResetRequest struct {
	Email     string     `json:"email"`
	RequestIP netip.Addr `json:"requestIP"`
}

var RequestPasswordReset = Define(ActionDef[PasswordResetRequest, Empty]{Name: "request_password_reset", Description: "Request a password reset email", Metadata: ActionMetadata{Public: true, Mutating: true}, Handler: func(ctx context.Context, c *Core, p PasswordResetRequest) (Empty, error) {
	if p.Email == "" {
		return Empty{}, invalid("Error decoding request: missing email")
	}
	token, err := genLostPasswordToken()
	if err != nil {
		return Empty{}, err
	}
	attrs := map[string]any{"request_time": time.Now(), "token": token, "email": p.Email}
	if p.RequestIP.IsValid() {
		attrs["request_ip"] = p.RequestIP
	}
	u, err := data.SelectUserByEmail(ctx, c.pool, p.Email)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return Empty{}, err
	}
	if u != nil {
		attrs["user_id"] = u.ID
	}
	if err := pgxutil.InsertRow(ctx, c.pool, "password_resets", attrs); err != nil {
		return Empty{}, err
	}
	if u == nil {
		c.logger.Warn("Password reset requested for missing email", "email", p.Email)
		return Empty{}, nil
	}
	if c.mailer == nil {
		return Empty{}, errors.New("mail is not configured")
	}
	return Empty{}, c.mailer.SendPasswordResetMail(p.Email, token)
}})

type PasswordReset struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

var ResetPassword = Define(ActionDef[PasswordReset, Session]{Name: "reset_password", Description: "Reset a password using an emailed token", Metadata: ActionMetadata{Public: true, Mutating: true}, Handler: func(ctx context.Context, c *Core, p PasswordReset) (Session, error) {
	reset, err := data.SelectPasswordResetByPK(ctx, c.pool, p.Token)
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, err
	}
	if !reset.UserID.Valid || reset.CompletionTime.Valid {
		return Session{}, ErrNotFound
	}
	u, err := data.SelectUserByPK(ctx, c.pool, reset.UserID.Int32)
	if err != nil {
		return Session{}, err
	}
	if err := SetPassword(u, p.Password); err != nil {
		return Session{}, err
	}
	if err := data.UpdateUser(ctx, c.pool, u.ID.Int32, u); err != nil {
		return Session{}, err
	}
	return c.session(ctx, u)
}})
