package identity

import (
	"net/mail"
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

func validateSex(sex int) error {
	if sex < 0 || sex > 2 {
		return invalid("性别不合法")
	}
	return nil
}

func validateName(name string) error {
	length := utf8.RuneCountInString(strings.TrimSpace(name))
	if length < 1 || length > 30 {
		return invalid("名称长度必须在 1 到 30 个字符之间")
	}
	return nil
}

func validateUsername(username string) error {
	if len(username) < 3 || len(username) > 30 || !isCode(username, true, true, true) {
		return invalid("用户名需为 3 到 30 位字母、数字、点、下划线或中划线")
	}
	return nil
}

func validateOptionalText(value string, limit int, message string) error {
	if utf8.RuneCountInString(value) > limit {
		return invalid(message)
	}
	return nil
}

func validateRemark(remark string) error {
	if utf8.RuneCountInString(remark) > 500 {
		return invalid("备注最长 500 个字符")
	}
	return nil
}

func validateAvatar(avatar string) error {
	if len(avatar) > 512 {
		return invalid("头像地址最长 512 个字符")
	}
	return nil
}

func validateAccountCount(count int) error {
	if count < 0 {
		return invalid("账号数量不能小于 0")
	}
	return nil
}

func validateTenantCode(code string) error {
	if len(code) < 2 || len(code) > 64 || !isCode(code, false, false, false) {
		return invalid("租户编码需为 2 到 64 位小写字母、数字或中划线")
	}
	return nil
}

func validatePostName(name string) error {
	length := utf8.RuneCountInString(strings.TrimSpace(name))
	if length < 1 || length > 50 {
		return invalid("岗位名称长度必须在 1 到 50 个字符之间")
	}
	return nil
}

func validatePostCode(code string) error {
	if len(code) < 1 || len(code) > 64 || !isCode(code, true, false, true) {
		return invalid("岗位编码需为 1 到 64 位字母、数字、下划线或中划线")
	}
	return nil
}

func validateRoleName(name string) error {
	length := utf8.RuneCountInString(strings.TrimSpace(name))
	if length < 1 || length > 30 {
		return invalid("角色名称长度必须在 1 到 30 个字符之间")
	}
	return nil
}

func validateRoleCode(code string) error {
	if len(code) < 1 || len(code) > 100 || !isPermissionCode(code) {
		return invalid("角色权限字符串需为 1 到 100 位字母、数字、冒号、下划线或中划线")
	}
	return nil
}

func validateRoleType(roleType int) error {
	if roleType != RoleTypeSystem && roleType != RoleTypeCustom {
		return invalid("角色类型必须是 1 或 2")
	}
	return nil
}

func isPermissionCode(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == ':' || r == '_' || r == '-':
		default:
			return false
		}
	}
	return true
}

func validateMenuName(name string) error {
	length := utf8.RuneCountInString(strings.TrimSpace(name))
	if length < 1 || length > 50 {
		return invalid("菜单名称长度必须在 1 到 50 个字符之间")
	}
	return nil
}

func validateMenuType(menuType int) error {
	if menuType != MenuTypeDirectory && menuType != MenuTypeMenu && menuType != MenuTypeButton {
		return invalid("菜单类型必须是 1、2 或 3")
	}
	return nil
}

func validateOrganizationCode(code string) error {
	if len(code) < 1 || len(code) > 64 || !isCode(code, true, false, true) {
		return invalid("部门编码需为 1 到 64 位字母、数字、下划线或中划线")
	}
	return nil
}

func validateEmail(email string) error {
	if email == "" {
		return nil
	}
	if len(email) > 50 {
		return invalid("邮箱格式不正确")
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return invalid("邮箱格式不正确")
	}
	return nil
}

func validateMobile(mobile string) error {
	if mobile == "" {
		return nil
	}
	if len(mobile) < 6 || len(mobile) > 11 {
		return invalid("手机号需为 6 到 11 位数字")
	}
	for _, r := range mobile {
		if r < '0' || r > '9' {
			return invalid("手机号需为 6 到 11 位数字")
		}
	}
	return nil
}

func validateAssignments(input AssignmentInput) error {
	if len(input.OrganizationIDs) == 0 {
		return invalid("至少选择一个任职组织")
	}
	seen := map[string]struct{}{}
	primarySeen := false
	for _, id := range input.OrganizationIDs {
		if id == "" {
			return invalid("部门标识不合法")
		}
		if _, ok := seen[id]; ok {
			return invalid("任职部门重复")
		}
		seen[id] = struct{}{}
		if id == input.PrimaryOrganizationID {
			primarySeen = true
		}
	}
	if !primarySeen {
		return invalid("主部门必须是任职部门之一")
	}
	return nil
}

func menuParentCreatesCycle(menus []Menu, id, parentID string) bool {
	parents := make(map[string]string, len(menus))
	for _, menu := range menus {
		parents[menu.ID] = menu.ParentID
	}
	return parentChainCycles(parents, id, parentID)
}

func parentCreatesCycle(orgs []Organization, id, parentID string) bool {
	parents := make(map[string]string, len(orgs))
	for _, org := range orgs {
		parents[org.ID] = org.ParentID
	}
	return parentChainCycles(parents, id, parentID)
}

func parentChainCycles(parents map[string]string, id, parentID string) bool {
	if parentID == "" {
		return false
	}
	if parentID == id {
		return true
	}
	seen := map[string]struct{}{}
	current := parentID
	for current != "" {
		if current == id {
			return true
		}
		if _, ok := seen[current]; ok {
			return true
		}
		seen[current] = struct{}{}
		next, ok := parents[current]
		if !ok {
			return false
		}
		current = next
	}
	return false
}

func isCode(value string, allowUpper, allowDot, allowUnderscore bool) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
		case allowUpper && r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-':
		case allowUnderscore && r == '_':
		case allowDot && r == '.':
		default:
			return false
		}
	}
	return true
}
