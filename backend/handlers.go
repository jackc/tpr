package backend

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"time"
	"uuid"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/tpr/backend/core"
)

// The adapters own wire decoding, status codes and serialization. All application
// work goes through the same core dispatcher as in-process callers.
func call[P, R any](w http.ResponseWriter, r *http.Request, e *environment, a *core.Action[P, R], p P) (R, bool) {
	c := e.core
	if c == nil {
		c = core.New(e.pool, e.mailer, e.logger)
	}
	result, err := a.Call(core.WithUser(r.Context(), e.user), c, p)
	if err == nil {
		return result, true
	}
	var input *core.InputError
	switch {
	case errors.Is(err, core.ErrUnauthenticated):
		w.WriteHeader(403)
		fmt.Fprint(w, "Bad or missing X-Authentication header")
	case errors.Is(err, core.ErrNotFound):
		if a.Name() == core.ResetPassword.Name() {
			w.WriteHeader(404)
		} else {
			http.NotFound(w, r)
		}
	case errors.As(err, &input):
		w.WriteHeader(422)
		if a.Name() == core.RequestPasswordReset.Name() {
			fmt.Fprint(w, input.Message)
		} else {
			fmt.Fprintln(w, input.Message)
		}
	default:
		if e.logger != nil {
			e.logger.Error("action failed", "action", a.Name(), "error", err)
		}
		http.Error(w, "Internal server error", 500)
	}
	return result, false
}
func decode[P any](w http.ResponseWriter, r *http.Request) (P, bool) {
	var p P
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		w.WriteHeader(422)
		fmt.Fprintf(w, "Error decoding request: %v", err)
		return p, false
	}
	return p, true
}
func respond(w http.ResponseWriter, status int, result any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(result)
}
func jsonAction[P, R any](w http.ResponseWriter, r *http.Request, e *environment, a *core.Action[P, R], status int) {
	p, ok := decode[P](w, r)
	if !ok {
		return
	}
	result, ok := call(w, r, e, a, p)
	if !ok {
		return
	}
	if _, empty := any(result).(core.Empty); empty {
		w.WriteHeader(status)
	} else {
		respond(w, status, result)
	}
}
func RegisterHandler(w http.ResponseWriter, r *http.Request, e *environment) {
	jsonAction(w, r, e, core.Register, 201)
}
func CreateSessionHandler(w http.ResponseWriter, r *http.Request, e *environment) {
	jsonAction(w, r, e, core.CreateSession, 201)
}
func CreateSubscriptionHandler(w http.ResponseWriter, r *http.Request, e *environment) {
	jsonAction(w, r, e, core.CreateSubscription, 201)
}
func UpdateAccountHandler(w http.ResponseWriter, r *http.Request, e *environment) {
	jsonAction(w, r, e, core.UpdateAccount, 200)
}
func ResetPasswordHandler(w http.ResponseWriter, r *http.Request, e *environment) {
	jsonAction(w, r, e, core.ResetPassword, 200)
}
func MarkMultipleItemsReadHandler(w http.ResponseWriter, r *http.Request, e *environment) {
	jsonAction(w, r, e, core.MarkMultipleItemsRead, 200)
}
func GetAccountHandler(w http.ResponseWriter, r *http.Request, e *environment) {
	if v, ok := call(w, r, e, core.GetAccount, core.Empty{}); ok {
		respond(w, 200, v)
	}
}
func GetFeedsHandler(w http.ResponseWriter, r *http.Request, e *environment) {
	rawAction(w, r, e, core.GetFeeds)
}
func GetUnreadItemsHandler(w http.ResponseWriter, r *http.Request, e *environment) {
	rawAction(w, r, e, core.GetUnreadItems)
}
func GetArchivedItemsHandler(w http.ResponseWriter, r *http.Request, e *environment) {
	rawAction(w, r, e, core.GetArchivedItems)
}
func rawAction(w http.ResponseWriter, r *http.Request, e *environment, a *core.Action[core.Empty, json.RawMessage]) {
	if v, ok := call(w, r, e, a, core.Empty{}); ok {
		w.Header().Set("Content-Type", "application/json")
		w.Write(v)
	}
}
func idAction(w http.ResponseWriter, r *http.Request, e *environment, a *core.Action[core.ID, core.Empty]) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	call(w, r, e, a, core.ID{ID: id.String()})
}
func DeleteSubscriptionHandler(w http.ResponseWriter, r *http.Request, e *environment) {
	idAction(w, r, e, core.DeleteSubscription)
}
func MarkItemReadHandler(w http.ResponseWriter, r *http.Request, e *environment) {
	idAction(w, r, e, core.MarkItemRead)
}
func DeleteSessionHandler(w http.ResponseWriter, r *http.Request, e *environment) {
	if _, ok := call(w, r, e, core.DeleteSession, core.SessionID{ID: chi.URLParam(r, "id")}); ok {
		http.SetCookie(w, &http.Cookie{Name: "sessionId", Value: "logged out", Expires: time.Unix(0, 0)})
	}
}
func RequestPasswordResetHandler(w http.ResponseWriter, r *http.Request, e *environment) {
	body, ok := decode[struct {
		Email string `json:"email"`
	}](w, r)
	if !ok {
		return
	}
	p := core.PasswordResetRequest{Email: body.Email}
	// Only the transport can supply the request address; ignore any JSON value.
	p.RequestIP = netip.Addr{}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		p.RequestIP, _ = netip.ParseAddr(host)
	}
	call(w, r, e, core.RequestPasswordReset, p)
}
func ImportFeedsHandler(w http.ResponseWriter, r *http.Request, e *environment) {
	file, _, err := r.FormFile("file")
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	if err != nil {
		w.WriteHeader(422)
		fmt.Fprintln(w, "No uploaded file found")
		return
	}
	defer file.Close()
	var doc core.OpmlDocument
	if err := xml.NewDecoder(file).Decode(&doc); err != nil {
		w.WriteHeader(422)
		fmt.Fprintln(w, "Error parsing OPML upload")
		return
	}
	if v, ok := call(w, r, e, core.ImportFeeds, doc); ok {
		respond(w, 200, v)
	}
}
func ExportFeedsHandler(w http.ResponseWriter, r *http.Request, e *environment) {
	if doc, ok := call(w, r, e, core.ExportFeeds, core.Empty{}); ok {
		w.Header().Set("Content-Type", "application/xml")
		w.Header().Set("Content-Disposition", `attachment; filename="opml.xml"`)
		fmt.Fprint(w, xml.Header)
		xml.NewEncoder(w).Encode(doc)
	}
}
