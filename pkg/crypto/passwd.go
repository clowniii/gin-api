package crypto

import (
	"crypto/md5"
	"encoding/hex"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Legacy MD5 helpers; 新算法使用 bcrypt

func MD5Hex(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

func IsLegacyMD5(s string) bool {
	return len(s) == 32 && !strings.HasPrefix(s, "{bcrypt}")
}

const bcryptCost = bcrypt.DefaultCost // 可根据需要调整

// HashPassword 生成 bcrypt 哈希（不再附加自定义前缀，长度保持 60 以内适配 varchar(64)）
func HashPassword(pwd string) string {
	bs, err := bcrypt.GenerateFromPassword([]byte(pwd), bcryptCost)
	if err != nil {
		return MD5Hex(pwd) // 兜底
	}
	return string(bs)
}

// VerifyPassword 自动检测算法（legacy md5 或 bcrypt $2 开头）
func VerifyPassword(plain, stored string) bool {
	if IsLegacyMD5(stored) { // 长度 32 且无 {bcrypt} 前缀视为旧 MD5
		return MD5Hex(plain) == stored
	}
	// 标准 bcrypt 格式以 $2a$ / $2b$ / $2y$ 开头
	if strings.HasPrefix(stored, "$2a$") || strings.HasPrefix(stored, "$2b$") || strings.HasPrefix(stored, "$2y$") {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(plain)) == nil
	}
	return false
}
