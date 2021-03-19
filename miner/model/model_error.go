/*
 * HighLoad Cup 2021
 *
 * ## Usage ## List of all custom errors First number is HTTP Status code, second is value of \"code\" field in returned JSON object, text description may or may not match \"message\" field in returned JSON object. - 422.1000: wrong coordinates - 422.1001: wrong depth - 409.1002: no more active licenses allowed - 409.1003: treasure is not digged
 *
 * API version: 1.0.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package model

// ModelError This model should match output of errors returned by go-swagger (like failed validation), to ensure our handlers use same format.
type ModelError struct {
	// Either same as HTTP Status Code OR >= 600 with HTTP Status Code 422
	Code    int32  `json:"code"`
	Message string `json:"message"`
}