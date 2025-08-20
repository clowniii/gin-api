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

const bcryptPrefix = "{bcrypt}"
const bcryptCost = bcrypt.DefaultCost // 可根据需要调整

// HashPassword 生成 bcrypt 哈希（带前缀用于区分）
func HashPassword(pwd string) string {
	bs, err := bcrypt.GenerateFromPassword([]byte(pwd), bcryptCost)
	if err != nil {
		// 兜底返回 MD5（极少数错误场景）
		return MD5Hex(pwd)
	}
	return bcryptPrefix + string(bs)
}

// VerifyPassword 自动检测算法（bcrypt / legacy md5）
func VerifyPassword(plain, stored string) bool {
	if IsLegacyMD5(stored) {
		return MD5Hex(plain) == stored
	}
	if strings.HasPrefix(stored, bcryptPrefix) {
		return bcrypt.CompareHashAndPassword([]byte(stored[len(bcryptPrefix):]), []byte(plain)) == nil
	}
	return false
}
