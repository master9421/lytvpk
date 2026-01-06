package parser

import (
	"strings"

	"git.lubar.me/ben/valve/vpk"
)

// ProcessWeaponVPK 处理武器类型VPK
func ProcessWeaponVPK(archive *vpk.Archive, vpkFile *VPKFile, secondaryTags map[string]bool) {
	vpkFile.PrimaryTag = "武器"

	// 遍历文件，检测具体武器
	for _, file := range archive.Files {
		filename := file.Name()

		// 检查是否为武器文件
		lowerFilename := strings.ToLower(filename)
		if strings.HasSuffix(lowerFilename, ".mdl") ||
			strings.HasSuffix(lowerFilename, ".vmt") ||
			strings.HasSuffix(lowerFilename, ".vtf") ||
			strings.HasSuffix(lowerFilename, ".wav") ||
			strings.HasSuffix(lowerFilename, ".mp3") {
			DetectWeaponType(filename, secondaryTags)
			if len(secondaryTags) > 0 {
				return
			}
		}
	}
}

// DetectWeaponType 检测武器类型
func DetectWeaponType(filename string, secondaryTags map[string]bool) {
	lowerFilename := strings.ToLower(filename)

	// Left 4 Dead 2 特定武器检测（最高优先级）
	l4d2SpecificWeapons := map[string]string{
		// 主武器 - 步枪
		"ak47":         "AK47",
		"desert":       "三连发",
		"desert_rifle": "三连发",
		"m16":          "M16",
		"m16a2":        "M16",
		"sg552":        "sg552",
		"m60":          "M60",

		// 狙击枪
		"sniper_awp":      "大狙",
		"sniper_military": "军狙",
		"sniper_a":        "军狙",
		"hunting_rifle":   "猎枪",
		"w_sniper_mini14": "猎枪",
		"sniper_scout":    "鸟狙",

		// 霰弹枪
		"chrome":      "铁喷",
		"m1014":       "铁喷",
		"w_shotgun":   "木喷",
		"autoshotgun": "一代连喷",
		"spas":        "二代连喷",

		// 冲锋枪
		"uzi":          "乌兹",
		"smg_a":        "消音",
		"smg_silenced": "消音",
		"mp5":          "MP5",

		// 手枪
		"magnum":         "马格南",
		"w_desert_eagle": "马格南",
		"pistol_glock":   "小手枪",
		"w_pistol_glock": "小手枪",
		"w_pistol_b":     "小手枪",

		// 投掷物
		"grenade_launcher": "榴弹",

		// 近战武器
		"machete":         "砍刀",
		"katana":          "武士刀",
		"baseball_bat":    "棒球棍",
		"w_bat":           "棒球棍",
		"knife":           "匕首",
		"chainsaw":        "电锯",
		"crowbar":         "撬棍",
		"fireaxe":         "消防斧",
		"frying_pan":      "平底锅",
		"electric_guitar": "吉他",
		"w_guitar":        "吉他",
		"cricket_bat":     "板球拍",
		"tonfa":           "警棍",
		"golf_club":       "高尔夫球杆",
		"shovel":          "铁铲",
		"pitchfork":       "草叉",
	}

	// 检测L4D2特定武器
	for keyword, weaponCode := range l4d2SpecificWeapons {
		if strings.Contains(lowerFilename, keyword) {
			secondaryTags[weaponCode] = true
			return
		}
	}
}
