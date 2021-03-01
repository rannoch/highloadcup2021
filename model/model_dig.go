/*
 * HighLoad Cup 2021
 *
 * ## Usage ## List of all custom errors First number is HTTP Status code, second is value of \"code\" field in returned JSON object, text description may or may not match \"message\" field in returned JSON object. - 422.1000: wrong coordinates - 422.1001: wrong depth - 409.1002: no more active licenses allowed - 409.1003: treasure is not digged
 *
 * API version: 1.0.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package model

// Dig struct for Dig
type Dig struct {
	// ID of the license this request is attached to.
	LicenseID int32 `json:"licenseID"`
	PosX      int32 `json:"posX"`
	PosY      int32 `json:"posY"`
	Depth     int32 `json:"depth"`
}