package response

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// BindJSON decodes a JSON object and writes a client-safe error on failure.
func BindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		writeBindError(c, dst, err)
		return false
	}
	return true
}

func writeBindError(c *gin.Context, dst any, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) || strings.Contains(err.Error(), "request body too large") {
		Fail(c, http.StatusRequestEntityTooLarge, CodePayloadTooLarge, "请求体过大", nil)
		return
	}

	var syntax *json.SyntaxError
	var unmarshal *json.UnmarshalTypeError
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.As(err, &syntax) || errors.As(err, &unmarshal) {
		Fail(c, http.StatusBadRequest, CodeBadRequest, "请求体不是合法的 JSON", nil)
		return
	}

	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		fields := make([]FieldError, 0, len(validationErrors))
		for _, fieldErr := range validationErrors {
			name := jsonFieldName(dst, fieldErr.StructField())
			fields = append(fields, FieldError{
				Field:   name,
				Message: fieldMessage(name, fieldErr.Tag()),
			})
		}
		InvalidParams(c, fields)
		return
	}

	Fail(c, http.StatusBadRequest, CodeBadRequest, "请求参数不合法", nil)
}

func fieldMessage(field, tag string) string {
	switch tag {
	case "required":
		return field + "为必填项"
	default:
		return field + "不合法"
	}
}

func jsonFieldName(dst any, structField string) string {
	valueType := reflect.TypeOf(dst)
	if valueType == nil {
		return structField
	}
	if valueType.Kind() == reflect.Pointer {
		valueType = valueType.Elem()
	}
	if valueType.Kind() != reflect.Struct {
		return structField
	}
	field, ok := valueType.FieldByName(structField)
	if !ok {
		return structField
	}
	tag := strings.Split(field.Tag.Get("json"), ",")[0]
	if tag == "" || tag == "-" {
		return structField
	}
	return tag
}
