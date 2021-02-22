/*
 * John Shields
 * Horton
 * API version: 1.0.0
 *
 * Error model for sending error messages to client.
 *
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package models

type Error struct {
	Code int32 `json:"code"`

	Messages string `json:"messages"`
}
