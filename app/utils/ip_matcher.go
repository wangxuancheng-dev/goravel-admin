package utils

import (
	"fmt"
	"net"
	"strings"

	"github.com/goravel/framework/support/str"

	apperrors "goravel/app/errors"
)

// IsIPInBlacklist 检查IP是否在黑名单中
// ip: 要检查的IP地址
// blacklistIPs: 黑名单IP字符串，支持：
//   - 单个IP: 192.168.1.1
//   - 多个IP（逗号分隔）: 192.168.1.1,192.168.1.2
//   - CIDR格式: 192.168.0.0/24
//   - IP范围: 192.168.1.1-192.168.1.100
func IsIPInBlacklist(ip string, blacklistIPs string) bool {
	if blacklistIPs == "" {
		return false
	}

	// 解析IP地址
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}
	matchIP := canonicalMatchIP(parsedIP)

	// 分割多个IP（支持逗号分隔）
	ipList := strings.SplitSeq(blacklistIPs, ",")
	for blacklistIP := range ipList {
		blacklistIP = str.Of(blacklistIP).Trim().String()
		if str.Of(blacklistIP).IsEmpty() {
			continue
		}

		// 检查单个IP（含 ::1 / 127.0.0.1 回环等价）
		if patternIP := net.ParseIP(blacklistIP); patternIP != nil {
			if canonicalMatchIP(patternIP).Equal(matchIP) {
				return true
			}
		} else if blacklistIP == ip {
			return true
		}

		// 检查CIDR格式 (192.168.0.0/24)
		if str.Of(blacklistIP).Contains("/") {
			_, ipNet, err := net.ParseCIDR(blacklistIP)
			if err == nil && (ipNet.Contains(parsedIP) || ipNet.Contains(matchIP)) {
				return true
			}
		}

		// 检查IP范围格式 (192.168.1.1-192.168.1.100)
		if str.Of(blacklistIP).Contains("-") {
			parts := str.Of(blacklistIP).Split("-")
			if len(parts) == 2 {
				startIP := net.ParseIP(str.Of(parts[0]).Trim().String())
				endIP := net.ParseIP(str.Of(parts[1]).Trim().String())
				if startIP != nil && endIP != nil {
					if isIPInRange(matchIP, startIP, endIP) || isIPInRange(parsedIP, startIP, endIP) {
						return true
					}
				}
			}
		}
	}

	return false
}

// canonicalMatchIP normalizes IPv4-mapped and IPv6 loopback to IPv4 for comparisons.
func canonicalMatchIP(ip net.IP) net.IP {
	if ip == nil {
		return nil
	}
	if v4 := ip.To4(); v4 != nil {
		return v4
	}
	if ip.IsLoopback() {
		return net.ParseIP("127.0.0.1").To4()
	}
	return ip
}

// isIPInRange 检查IP是否在指定范围内
func isIPInRange(ip, startIP, endIP net.IP) bool {
	ipBytes := ip.To4()
	startBytes := startIP.To4()
	endBytes := endIP.To4()

	if ipBytes == nil || startBytes == nil || endBytes == nil {
		return false
	}

	// 将IP地址转换为32位整数进行比较
	ipInt := uint32(ipBytes[0])<<24 | uint32(ipBytes[1])<<16 | uint32(ipBytes[2])<<8 | uint32(ipBytes[3])
	startInt := uint32(startBytes[0])<<24 | uint32(startBytes[1])<<16 | uint32(startBytes[2])<<8 | uint32(startBytes[3])
	endInt := uint32(endBytes[0])<<24 | uint32(endBytes[1])<<16 | uint32(endBytes[2])<<8 | uint32(endBytes[3])

	return ipInt >= startInt && ipInt <= endInt
}

// ValidateBlacklistIP 验证黑名单IP格式
// 返回业务错误类型，如果格式正确返回 nil
func ValidateBlacklistIP(ipStr string) error {
	if ipStr == "" {
		return apperrors.ErrIPAddressRequired
	}

	ipList := strings.SplitSeq(ipStr, ",")
	for ip := range ipList {
		ip = str.Of(ip).Trim().String()
		if str.Of(ip).IsEmpty() {
			continue
		}

		// 检查CIDR格式
		if str.Of(ip).Contains("/") {
			_, _, err := net.ParseCIDR(ip)
			if err != nil {
				return apperrors.ErrInvalidCIDRFormat.WithMessage(fmt.Sprintf("CIDR格式错误: %s", ip))
			}
			continue
		}

		// 检查IP范围格式
		if str.Of(ip).Contains("-") {
			parts := str.Of(ip).Split("-")
			if len(parts) != 2 {
				return apperrors.ErrInvalidIPRangeFormat.WithMessage(fmt.Sprintf("IP范围格式错误: %s (格式应为: 192.168.1.1-192.168.1.100)", ip))
			}
			startIP := net.ParseIP(str.Of(parts[0]).Trim().String())
			endIP := net.ParseIP(str.Of(parts[1]).Trim().String())
			if startIP == nil {
				return apperrors.ErrInvalidIPFormat.WithMessage(fmt.Sprintf("起始IP格式错误: %s", parts[0]))
			}
			if endIP == nil {
				return apperrors.ErrInvalidIPFormat.WithMessage(fmt.Sprintf("结束IP格式错误: %s", parts[1]))
			}
			// 验证范围是否有效（结束IP应该大于等于起始IP）
			if !isIPInRange(endIP, startIP, endIP) && !endIP.Equal(startIP) {
				// 简单比较：检查结束IP是否大于起始IP
				startBytes := startIP.To4()
				endBytes := endIP.To4()
				if startBytes != nil && endBytes != nil {
					for i := range 4 {
						if endBytes[i] < startBytes[i] {
							return apperrors.ErrInvalidIPRangeOrder.WithMessage(fmt.Sprintf("结束IP必须大于等于起始IP: %s", ip))
						}
						if endBytes[i] > startBytes[i] {
							break
						}
					}
				}
			}
			continue
		}

		// 检查单个IP格式
		parsedIP := net.ParseIP(ip)
		if parsedIP == nil {
			return apperrors.ErrInvalidIPFormat.WithMessage(fmt.Sprintf("IP地址格式错误: %s", ip))
		}
	}

	return nil
}

// FormatIPRange 格式化IP范围显示
func FormatIPRange(ipStr string) string {
	if str.Of(ipStr).Contains("-") {
		parts := str.Of(ipStr).Split("-")
		if len(parts) == 2 {
			return fmt.Sprintf("%s ~ %s", str.Of(parts[0]).Trim().String(), str.Of(parts[1]).Trim().String())
		}
	}
	return ipStr
}
