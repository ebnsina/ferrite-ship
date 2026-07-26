package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ebnsina/ferrite-ship/internal/apierr"
	"github.com/ebnsina/ferrite-ship/internal/store"
)

// A handler that has already chosen an entry from the catalogue must have that
// choice respected.
//
// This went wrong in exactly one way: ToolNotInstalled wraps store.ErrNotFound
// as its cause, so the generic "is this a missing row?" check matched and
// replaced the specific message with "We could not find that." The person was
// then told nothing about what was actually missing.
func TestAChosenErrorIsNotReplacedByAGenericOne(t *testing.T) {
	api := &API{log: slog.New(slog.DiscardHandler)}

	recorder := httptest.NewRecorder()
	api.fail(recorder, apierr.ToolNotInstalled.WithCause(store.ErrNotFound))

	var body errorBody
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if body.Message != apierr.ToolNotInstalled.Message {
		t.Errorf("message is %q, want the one the handler chose: %q",
			body.Message, apierr.ToolNotInstalled.Message)
	}
	if body.Action != apierr.ToolNotInstalled.Action {
		t.Errorf("action is %q, want %q", body.Action, apierr.ToolNotInstalled.Action)
	}
	if recorder.Code != apierr.ToolNotInstalled.Status {
		t.Errorf("status is %d, want %d", recorder.Code, apierr.ToolNotInstalled.Status)
	}
}

// A bare store error still gets the wording of whatever the route is about,
// which is what lets one handler say "server" and another say "job".
func TestAnUnclassifiedMissingRowUsesTheRoutesWording(t *testing.T) {
	api := &API{log: slog.New(slog.DiscardHandler)}

	recorder := httptest.NewRecorder()
	api.failServer(recorder, store.ErrNotFound)

	var body errorBody
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if body.Message != apierr.ServerNotFound.Message {
		t.Errorf("message is %q, want %q", body.Message, apierr.ServerNotFound.Message)
	}
	if recorder.Code != http.StatusNotFound {
		t.Errorf("status is %d, want 404", recorder.Code)
	}
}

// Every response the browser sees carries something to show and something to
// do next. An entry with an empty message would render as a blank error box.
func TestEveryFailureSaysSomething(t *testing.T) {
	api := &API{log: slog.New(slog.DiscardHandler)}

	for _, err := range []*apierr.Error{
		apierr.UnknownTool, apierr.ToolNotInstalled,
		apierr.ToolNeedsAddress, apierr.ToolNotReady,
	} {
		recorder := httptest.NewRecorder()
		api.fail(recorder, err)

		var body errorBody
		if decodeErr := json.NewDecoder(recorder.Body).Decode(&body); decodeErr != nil {
			t.Fatalf("decode: %v", decodeErr)
		}
		if body.Message == "" || body.Action == "" {
			t.Errorf("%s: message=%q action=%q; both must be filled in",
				body.Code, body.Message, body.Action)
		}
		if body.RequestID == "" {
			t.Errorf("%s: no request id, so a report of this cannot be traced", body.Code)
		}
	}
}
