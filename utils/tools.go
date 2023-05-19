package utils

import (
	"crypto/md5"
	sha12 "crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"strings"

	"gin-app/config"
	"gin-app/models/admin"
)

func IP2Long(ip string) int64 {
	// 将 IP 地址解析为 net.IP 类型
	netIP := net.ParseIP(ip)
	if netIP == nil {
		return 0
	}
	// 将 IPv4 地址转换为 4 字节整数
	ipBytes := netIP.To4()
	if ipBytes == nil {
		return 0
	}
	// 将 4 字节整数转换为 uint32 类型的 ip2long 值
	return int64(binary.BigEndian.Uint32(ipBytes))
}

func GenerateMenuTree(menus []*admin.AdminMenu, need bool) []*admin.AdminMenu {
	// 定义一个 map，用来存储父节点和其下的子节点
	nodes := make(map[int64][]*admin.AdminMenu)
	// 遍历菜单列表，将菜单按照父节点归类
	for _, menu := range menus {
		nodes[menu.Fid] = append(nodes[menu.Fid], menu)
	}
	// 定义一个递归函数，用来将一个节点下的子节点生成树状结构
	var buildTree func(int64) []*admin.AdminMenu
	buildTree = func(fid int64) []*admin.AdminMenu {
		var tree []*admin.AdminMenu
		// 获取 fid 下的所有子节点
		children := nodes[fid]
		for _, child := range children {
			// 递归生成子节点的树状结构
			if need {
				if child.Router != "" {
					child.Children = buildTree(child.ID)
					// 将子节点加入树中
					tree = append(tree, child)
				}
			} else {
				child.Children = buildTree(child.ID)
				// 将子节点加入树中
				tree = append(tree, child)
			}
		}
		return tree
	}
	// 从顶级菜单开始递归生成树状结构
	return buildTree(0)
}
func UserMd5(pwd string) string {
	sha1 := sha12.Sum([]byte(pwd))
	str := md5.Sum([]byte(hex.EncodeToString(sha1[:]) + config.Conf.AuthKey))
	return fmt.Sprintf("%x", str)
}
func Md5(arg ...string) string {
	var str string
	for _, s := range arg {
		str += s
	}
	str2 := md5.Sum([]byte(str))
	return fmt.Sprintf("%x", str2)
}

func ToSnakeCase(str string) string {
	var builder strings.Builder
	for i, char := range str {
		if char >= 'A' && char <= 'Z' {
			if i > 0 {
				builder.WriteRune('_')
			}
			builder.WriteRune(char + ('a' - 'A'))
		} else {
			builder.WriteRune(char)
		}
	}
	return builder.String()
}
