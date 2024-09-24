//
// @package Showcase-SigNoz-Golang
//
// @file Todo tests for gorm
// @copyright 2023-present Christoph Kappel <christoph@unexist.dev>
// @version $Id$
//
// This program can be distributed under the terms of the Apache License v2.0.
// See the file LICENSE for details.
//

//go:build gorm

package test

import (
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"os"
	"testing"

	"github.com/unexist/showcase-microservices-golang/adapter"
	"github.com/unexist/showcase-microservices-golang/domain"
	"github.com/unexist/showcase-microservices-golang/infrastructure"

	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
)

/* Test globals */
var engine *gin.Engine
var todoRepository *infrastructure.TodoGormRepository

func TestMain(m *testing.M) {
	/* Create database connection */
	todoRepository = infrastructure.NewTodoGormRepository()

	connectionString :=
		fmt.Sprintf("host=localhost user=%s password=%s dbname=%s sslmode=disable",
			os.Getenv("TEST_DB_USERNAME"),
			os.Getenv("TEST_DB_PASSWORD"),
			os.Getenv("TEST_DB_NAME"))

	err := todoRepository.Open(connectionString)

	if nil != err {
		log.Fatal(err)
	}

	defer todoRepository.Close()

	/* Create business stuff */
	todoService := domain.NewTodoService(todoRepository)
	idService := domain.NewIdService()
	todoResource := adapter.NewTodoResource(todoService, idService)

	/* Finally start Gin */
	engine = gin.Default()

	todoResource.RegisterRoutes(engine)

	retCode := m.Run()

	os.Exit(retCode)
}

func executeRequest(req *http.Request) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()

	engine.ServeHTTP(recorder, req)

	return recorder
}

func checkResponseCode(t *testing.T, expected, actual int) {
	assert.Equal(t, expected, actual, "Expected different response code")
}

func TestEmptyTable(t *testing.T) {
	todoRepository.Clear()

	req, _ := http.NewRequest("GET", "/todo", nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusOK, response.Code)

	if body := response.Body.String(); "[]" != body {
		t.Errorf("Expected an empty array. Got %s", body)
	}
}

func TestGetNonExistentTodo(t *testing.T) {
	todoRepository.Clear()

	req, _ := http.NewRequest("GET", "/todo/11", nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusNotFound, response.Code)

	var m map[string]string
	json.Unmarshal(response.Body.Bytes(), &m)

	assert.Equal(t, "Todo not found", m["error"],
		"Expected the 'error' key of the response to be set to 'Todo not found'")
}

func TestCreateTodo(t *testing.T) {
	todoRepository.Clear()

	var jsonStr = []byte(`{"title":"string", "description": "string"}`)

	req, _ := http.NewRequest("POST", "/todo", bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")

	response := executeRequest(req)
	checkResponseCode(t, http.StatusCreated, response.Code)

	var m map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &m)

	assert.Equal(t, 1.0, m["id"], "Expected todo ID to be '1'")
	assert.Equal(t, "string", m["title"], "Expected todo title to be 'string'")
	assert.Equal(t, "string", m["description"], "Expected todo description to be 'string'")
}

func TestGetTodo(t *testing.T) {
	todoRepository.Clear()
	addTodos(1)

	req, _ := http.NewRequest("GET", "/todo/1", nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusOK, response.Code)
}

func addTodos(count int) {
	if 1 > count {
		count = 1
	}

	todo := domain.Todo{}

	for i := 0; i < count; i++ {
		todo.ID = i
		todo.Title = "Todo " + strconv.Itoa(i)
		todo.Description = "string"

		todoRepository.CreateTodo(&todo)
	}
}

func TestUpdateTodo(t *testing.T) {
	todoRepository.Clear()
	addTodos(1)

	req, _ := http.NewRequest("GET", "/todo/1", nil)
	response := executeRequest(req)

	var origTodo map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &origTodo)

	var jsonStr = []byte(`{"title":"new string", "description": "new string"}`)

	req, _ = http.NewRequest("PUT", "/todo/1", bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")

	response = executeRequest(req)
	checkResponseCode(t, http.StatusOK, response.Code)

	var m map[string]interface{}
	json.Unmarshal(response.Body.Bytes(), &m)

	assert.Equal(t, origTodo["id"], m["id"], "Expected the id to remain the same")
	assert.NotEqual(t, origTodo["title"], m["title"], "Expected the title to change")
	assert.NotEqual(t, origTodo["description"], m["description"], "Expected the description to change")
}

func TestDeleteTodo(t *testing.T) {
	todoRepository.Clear()
	addTodos(1)

	req, _ := http.NewRequest("GET", "/todo/1", nil)
	response := executeRequest(req)
	checkResponseCode(t, http.StatusOK, response.Code)

	req, _ = http.NewRequest("DELETE", "/todo/1", nil)
	response = executeRequest(req)
	checkResponseCode(t, http.StatusNoContent, response.Code)

	req, _ = http.NewRequest("GET", "/todo/1", nil)
	response = executeRequest(req)
	checkResponseCode(t, http.StatusNotFound, response.Code)
}
