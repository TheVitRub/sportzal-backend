package service

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"workout-app/backend/internal/domain/apperrors"
	"workout-app/backend/internal/domain/model"
)

func ValidateMetricSchema(schema model.MetricSchema) error {
	if strings.TrimSpace(schema.Type) == "" {
		return apperrors.BadRequest("тип схемы метрик обязателен")
	}
	if len(schema.Fields) == 0 {
		return apperrors.BadRequest("добавьте хотя бы одну метрику")
	}

	seen := map[string]bool{}
	for _, field := range schema.Fields {
		key := strings.TrimSpace(field.Key)
		if key == "" {
			return apperrors.BadRequest("ключ метрики обязателен")
		}
		if seen[key] {
			return apperrors.BadRequest("метрика " + key + " дублируется")
		}
		seen[key] = true
		if strings.TrimSpace(field.Label) == "" {
			return apperrors.BadRequest("у метрики " + key + " нет названия")
		}
		switch field.ValueType {
		case "int", "float", "text":
		default:
			return apperrors.BadRequest("метрика " + key + " имеет неподдерживаемый тип")
		}
	}

	return nil
}

func NormalizeMetricValues(schema model.MetricSchema, input map[string]interface{}) (map[string]interface{}, error) {
	if input == nil {
		input = map[string]interface{}{}
	}

	values := map[string]interface{}{}
	for _, field := range schema.Fields {
		raw, exists := input[field.Key]
		if !exists || raw == nil || raw == "" {
			if field.Required {
				return nil, apperrors.BadRequest("поле " + field.Label + " обязательно")
			}
			continue
		}

		switch field.ValueType {
		case "int":
			n, err := asFloat(raw)
			if err != nil || math.Mod(n, 1) != 0 {
				return nil, apperrors.BadRequest("поле " + field.Label + " должно быть целым числом")
			}
			if err := checkBounds(field, n); err != nil {
				return nil, err
			}
			values[field.Key] = int64(n)
		case "float":
			n, err := asFloat(raw)
			if err != nil {
				return nil, apperrors.BadRequest("поле " + field.Label + " должно быть числом")
			}
			if err := checkBounds(field, n); err != nil {
				return nil, err
			}
			values[field.Key] = n
		case "text":
			text := strings.TrimSpace(strings.TrimSpace(toString(raw)))
			if text == "" && field.Required {
				return nil, apperrors.BadRequest("поле " + field.Label + " обязательно")
			}
			if text != "" {
				values[field.Key] = text
			}
		}
	}

	return values, nil
}

func checkBounds(field model.MetricField, value float64) error {
	if field.Min != nil && value < *field.Min {
		return apperrors.BadRequest("поле " + field.Label + " меньше минимума")
	}
	if field.Max != nil && value > *field.Max {
		return apperrors.BadRequest("поле " + field.Label + " больше максимума")
	}
	return nil
}

func asFloat(v interface{}) (float64, error) {
	switch value := v.(type) {
	case float64:
		return value, nil
	case float32:
		return float64(value), nil
	case int:
		return float64(value), nil
	case int64:
		return float64(value), nil
	case string:
		text := strings.ReplaceAll(strings.TrimSpace(value), ",", ".")
		return strconv.ParseFloat(text, 64)
	default:
		return 0, strconv.ErrSyntax
	}
}

func toString(v interface{}) string {
	switch value := v.(type) {
	case string:
		return value
	default:
		return fmt.Sprint(value)
	}
}
