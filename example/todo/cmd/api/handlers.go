package main

import (
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/core/command"
	"github.com/larsartmann/go-cqrs-lite/core/query"
	"github.com/larsartmann/go-cqrs-lite/example/todo/commands"
	"github.com/larsartmann/go-cqrs-lite/example/todo/domain"
	"github.com/larsartmann/go-cqrs-lite/example/todo/queries"
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
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": qErr.Error()})
			return
		}

		result, err := qDisp.Dispatch(r.Context(), q)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
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
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		aggregateID := newAggregateID()
		cmd, cmdErr := commands.NewCreateTodoCommand(
			aggregateID, req.Title, req.Description, req.Priority, req.Tags,
		)
		if cmdErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": cmdErr.Error()})
			return
		}

		if err := cmdDisp.Dispatch(r.Context(), cmd); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusCreated, map[string]string{"id": aggregateID.String()})
	}
}

func getTodo(qDisp *query.Dispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		todoID, err := domain.ParseTodoID(r.PathValue("id"))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
			return
		}

		q, qErr := queries.NewGetTodoQuery(todoID)
		if qErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": qErr.Error()})
			return
		}

		result, err := qDisp.Dispatch(r.Context(), q)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "todo not found"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"data": result})
	}
}

func updateTodo(cmdDisp *command.Dispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		aggregateID, err := parseAggregateID(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
			return
		}

		var req struct {
			Title       string `json:"title"`
			Description string `json:"description"`
		}

		if err := decodeJSON(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		cmd, cmdErr := commands.NewUpdateTodoCommand(aggregateID, req.Title, req.Description)
		if cmdErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": cmdErr.Error()})
			return
		}

		if err := cmdDisp.Dispatch(r.Context(), cmd); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, map[string]bool{"success": true})
	}
}

func deleteTodo(cmdDisp *command.Dispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		aggregateID, err := parseAggregateID(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
			return
		}

		cmd, cmdErr := commands.NewDeleteTodoCommand(aggregateID)
		if cmdErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": cmdErr.Error()})
			return
		}

		if err := cmdDisp.Dispatch(r.Context(), cmd); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func changeStatus(cmdDisp *command.Dispatcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		aggregateID, err := parseAggregateID(r)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
			return
		}

		var req struct {
			Status domain.TodoStatus `json:"status"`
		}

		if err := decodeJSON(r, &req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		cmd, cmdErr := commands.NewChangeStatusCommand(aggregateID, req.Status)
		if cmdErr != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": cmdErr.Error()})
			return
		}

		if err := cmdDisp.Dispatch(r.Context(), cmd); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
