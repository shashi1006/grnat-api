// Package response provides standardised Gin JSON response helpers.
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Envelope wraps all API responses.
type Envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// APIError is the standard error body.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Meta carries pagination metadata.
type Meta struct {
	Total  int64 `json:"total,omitempty"`
	Limit  int   `json:"limit,omitempty"`
	Offset int   `json:"offset,omitempty"`
}

// OK sends a 200 response with data.
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Envelope{Success: true, Data: data})
}

// OKWithMeta sends a 200 response with data and pagination meta.
func OKWithMeta(c *gin.Context, data interface{}, meta *Meta) {
	c.JSON(http.StatusOK, Envelope{Success: true, Data: data, Meta: meta})
}

// Created sends a 201 response.
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Envelope{Success: true, Data: data})
}

// NoContent sends a 204 response.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// BadRequest sends a 400 error response.
func BadRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, Envelope{Success: false, Error: &APIError{Code: "bad_request", Message: msg}})
}

// Unauthorized sends a 401 error response.
func Unauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, Envelope{Success: false, Error: &APIError{Code: "unauthorized", Message: msg}})
}

// Forbidden sends a 403 error response.
func Forbidden(c *gin.Context, msg string) {
	c.JSON(http.StatusForbidden, Envelope{Success: false, Error: &APIError{Code: "forbidden", Message: msg}})
}

// NotFound sends a 404 error response.
func NotFound(c *gin.Context, msg string) {
	c.JSON(http.StatusNotFound, Envelope{Success: false, Error: &APIError{Code: "not_found", Message: msg}})
}

// Conflict sends a 409 error response.
func Conflict(c *gin.Context, msg string) {
	c.JSON(http.StatusConflict, Envelope{Success: false, Error: &APIError{Code: "conflict", Message: msg}})
}

// UnprocessableEntity sends a 422 error response.
func UnprocessableEntity(c *gin.Context, msg string) {
	c.JSON(http.StatusUnprocessableEntity, Envelope{Success: false, Error: &APIError{Code: "unprocessable_entity", Message: msg}})
}

// TooManyRequests sends a 429 error response.
func TooManyRequests(c *gin.Context) {
	c.JSON(http.StatusTooManyRequests, Envelope{Success: false, Error: &APIError{Code: "rate_limit_exceeded", Message: "Too many requests — please slow down"}})
}

// InternalError sends a 500 error response.
func InternalError(c *gin.Context, err error) {
	msg := "An internal error occurred"
	if err != nil {
		msg = err.Error()
	}
	c.JSON(http.StatusInternalServerError, Envelope{Success: false, Error: &APIError{Code: "internal_error", Message: msg}})
}
