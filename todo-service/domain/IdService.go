//
// @package Showcase-SigNoz-Golang
//
// @file Id service
// @copyright 2024-present Christoph Kappel <christoph@unexist.dev>
// @version $Id$
//
// This program can be distributed under the terms of the Apache License v2.0.
// See the file LICENSE for details.
//

package domain

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"

	"github.com/unexist/showcase-microservices-golang/infrastructure/utils"
)

type IdService struct{}

type IdServiceReply struct {
	UUID string `json:"id"`
}

func NewIdService() *IdService {
	return &IdService{}
}

func (service *IdService) GetId(ctx *gin.Context) (string, error) {
	tracer := otel.GetTracerProvider().Tracer("todo-service")
	_, span := tracer.Start(ctx, "get-id")
	defer span.End()

	//response, err := otelhttp.Get(ctx, fmt.Sprintf("http://%s/id",
	//	utils.GetEnvOrDefault("APP_ID_HOST_PORT", "localhost:8081")))

	request, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("http://%s/id",
			utils.GetEnvOrDefault("APP_ID_HOST_PORT", "localhost:8081")),
		nil)
	request.Header.Set("CorrelationId", ctx.GetString("correlation_id"))

	// Include a tracing transport
	httpclient := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	response, err := httpclient.Do(request)

	if err != nil {
		return "", err
	}

	jsonBytes, _ := io.ReadAll(response.Body)

	var reply IdServiceReply

	err = json.Unmarshal(jsonBytes, &reply)

	if err != nil {
		return "", err
	}

	return reply.UUID, nil
}
