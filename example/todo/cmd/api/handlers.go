package main

import (
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/command/v3"
	cqrsevent "github.com/larsartmann/go-cqrs-lite/event/v3"
	"github.com/larsartmann/go-cqrs-lite/example/todo/commands"
	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
	"github.com/larsartmann/go-cqrs-lite/example/todo/queries"
	"github.com/larsartmann/go-cqrs-lite/query/v3"
)

func registerTodoRoutes(
	mux *http.ServeMux,
	cmdDisp *command.Dispatcher,
	queryDisp *query.Dispatcher,
) {
	mux.Handle("GET /api/v1/todos", listTodos(queryDisp))
	mux.Handle("POST /api/v1/todos", createTodo(cmdDisp))
	mux.Handle("GET /api/v1/todos/{id}", getTodo(queryDisp))
	mux.Handle("PUT /api/v1/todos/{id}", updateTodo(cmdDisp))
	mux.Handle("DELETE /api/v1/todos/{id}", deleteTodo(cmdDisp))
	mux.Handle("PATCH /api/v1/todos/{id}/status", changeStatus(cmdDisp))
}

func listTodos(qDisp *query.Dispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q, qErr := queries.NewListTodosQuery()
		if qErr != nil {
			writeError(w, http.StatusInternalServerError, qErr.Error())

			return
		}

		result, err := query.DispatchTyped[*queries.ListTodosResult](r.Context(), qDisp, q)
		if err != nil {
			writeDispatchError(w, err)

			return
		}

		writeJSON(w, http.StatusOK, result)
	}
}

func createTodo(cmdDisp *command.Dispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Title       string   `json:"title"`
			Description string   `json:"description"`
			Priority    int      `json:"priority"`
			Tags        []string `json:"tags"`
		}

		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())

			return
		}

		aggregateID := newAggregateID()

		cmd, cmdErr := commands.NewCreateTodoCommand(
			aggregateID, req.Title, req.Description, req.Priority, req.Tags,
		)
		if cmdErr != nil {
			writeError(w, http.StatusBadRequest, cmdErr.Error())

			return
		}

		dispatchAndRespond(w, r, cmdDisp, cmd, func() {
			writeJSON(w, http.StatusCreated, map[string]string{"id": aggregateID.String()})
		})
	}
}

func getTodo(qDisp *query.Dispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		todoID, err := domain.ParseTodoID(r.PathValue("id"))
		if err != nil {
			writeErrorInvalidID(w)

			return
		}

		q, qErr := queries.NewGetTodoQuery(todoID)
		if qErr != nil {
			writeError(w, http.StatusBadRequest, qErr.Error())

			return
		}

		result, err := query.DispatchTyped[*queries.GetTodoResult](r.Context(), qDisp, q)
		if err != nil {
			writeDispatchError(w, err)

			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"data": result})
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// writeDispatchError maps a dispatch error to the correct HTTP status code
// using the CQRS error taxonomy, avoiding leaking internal error text.
func writeDispatchError(w http.ResponseWriter, err error) {
	switch cqrsevent.Classify(err) {
	case cqrsevent.Rejection:
		writeError(w, http.StatusBadRequest, "request rejected")
	case cqrsevent.Conflict:
		writeError(w, http.StatusConflict, "conflict")
	case cqrsevent.Corruption:
		writeError(w, http.StatusInternalServerError, "data integrity error")
	case cqrsevent.Transient, cqrsevent.Infrastructure:
		writeError(w, http.StatusServiceUnavailable, "temporarily unavailable")
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func writeErrorInvalidID(w http.ResponseWriter) {
	writeError(w, http.StatusBadRequest, "invalid id")
}

func dispatchAndRespond(
	w http.ResponseWriter,
	r *http.Request,
	disp *command.Dispatcher,
	cmd command.Command,
	successFn func(),
) {
	if err := disp.Dispatch(r.Context(), cmd); err != nil {
		writeDispatchError(w, err)

		return
	}

	successFn()
}

func updateTodo(cmdDisp *command.Dispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		aggregateID, err := parseAggregateID(r)
		if err != nil {
			writeErrorInvalidID(w)

			return
		}

		var req struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		}

		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())

			return
		}

		cmd, cmdErr := commands.NewUpdateTodoCommand(aggregateID, req.Title, req.Description)
		if cmdErr != nil {
			writeError(w, http.StatusBadRequest, cmdErr.Error())

			return
		}

		dispatchAndRespond(w, r, cmdDisp, cmd, func() {
			writeJSON(w, http.StatusOK, map[string]bool{"success": true})
		})
	}
}

func deleteTodo(cmdDisp *command.Dispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		aggregateID, err := parseAggregateID(r)
		if err != nil {
			writeErrorInvalidID(w)

			return
		}

		cmd, cmdErr := commands.NewDeleteTodoCommand(aggregateID)
		if cmdErr != nil {
			writeError(w, http.StatusBadRequest, cmdErr.Error())

			return
		}

		dispatchAndRespond(w, r, cmdDisp, cmd, func() {
			w.WriteHeader(http.StatusNoContent)
		})
	}
}

func changeStatus(cmdDisp *command.Dispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		aggregateID, err := parseAggregateID(r)
		if err != nil {
			writeErrorInvalidID(w)

			return
		}

		var req struct {
			Status domain.TodoStatus `json:"status"`
		}

		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())

			return
		}

		cmd, cmdErr := commands.NewChangeStatusCommand(aggregateID, req.Status)
		if cmdErr != nil {
			writeError(w, http.StatusBadRequest, cmdErr.Error())

			return
		}

		dispatchAndRespond(w, r, cmdDisp, cmd, func() {
			w.WriteHeader(http.StatusNoContent)
		})
	}
}
