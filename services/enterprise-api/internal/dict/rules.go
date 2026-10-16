package dict

import (
	"strings"
	"unicode/utf8"
)

func normalizePage(page, pageSize int) (int, int, error) {
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 20
	}
	if page < 1 || pageSize < 1 || pageSize > 100 {
		return 0, 0, invalid("分页参数不合法")
	}
	return page, pageSize, nil
}

func validateStatus(status int) error {
	if status != StatusNormal && status != StatusDisabled {
		return invalid("状态不合法")
	}
	return nil
}

func validateDictName(name string) error {
	length := utf8.RuneCountInString(strings.TrimSpace(name))
	if length < 1 || length > 100 {
		return invalid("字典名称长度必须在 1 到 100 个字符之间")
	}
	return nil
}

func validateDictTypeCode(code string) error {
	if len(code) < 1 || len(code) > 100 || !isDictCode(code) {
		return invalid("字典类型需为 1 到 100 位字母、数字或下划线")
	}
	return nil
}

func validateLabel(label string) error {
	length := utf8.RuneCountInString(strings.TrimSpace(label))
	if length < 1 || length > 100 {
		return invalid("字典标签长度必须在 1 到 100 个字符之间")
	}
	return nil
}

func validateValue(value string) error {
	if len(value) < 1 || len(value) > 100 || !isDictValue(value) {
		return invalid("字典键值需为 1 到 100 位字母、数字、下划线、点或中划线")
	}
	return nil
}

func validateSort(sort int) error {
	if sort < 0 || sort > 1000000 {
		return invalid("排序值不合法")
	}
	return nil
}

func validateRemark(remark string) error {
	if utf8.RuneCountInString(remark) > 500 {
		return invalid("备注最长 500 个字符")
	}
	return nil
}

func validateColorType(color string) error {
	switch color {
	case "", "default", "primary", "success", "info", "warning", "danger":
		return nil
	default:
		return invalid("颜色类型不合法")
	}
}

func validateCSSClass(class string) error {
	if len(class) > 100 {
		return invalid("css 样式最长 100 个字符")
	}
	return nil
}

func validateActor(actor string) error {
	if utf8.RuneCountInString(actor) > 64 {
		return invalid("操作者标识最长 64 个字符")
	}
	return nil
}

func isDictCode(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return false
	}
	return true
}

func isDictValue(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			continue
		}
		return false
	}
	return true
}
