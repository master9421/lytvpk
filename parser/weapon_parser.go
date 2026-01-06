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
		if strings.Contains(lowerFilename, "models/weapons/") ||
			strings.Contains(lowerFilename, "sound/weapons/") ||
			strings.Contains(lowerFilename, "materials/models/weapons/") {
			DetectWeaponType(filename, secondaryTags)
		}
	}
}

// DetectWeaponType 检测武器类型 - 基于NekoVpk模式，返回武器代码
func DetectWeaponType(filename string, secondaryTags map[string]bool) {
	lowerFilename := strings.ToLower(filename)

	// Left 4 Dead 2 特定武器检测（最高优先级）
	l4d2SpecificWeapons := map[string]string{
		// 主武器 - 步枪
		"rifle_ak47":   "AK47",
		"rifle_desert": "三连发",
		"rifle_m16":    "M16",
		"rifle_m16a2":  "M16",
		"rifle_sg552":  "sg552",
		"rifle_m60":    "M60",

		// 狙击枪
		"sniper_awp":      "大狙",
		"sniper_military": "军狙",
		"hunting_rifle":   "猎枪",
		"w_sniper_mini14": "猎枪",
		"sniper_scout":    "鸟狙",

		// 霰弹枪
		"shotgun_pump":       "木喷",
		"pumpshotgun":        "木喷",
		"shotgun_chrome":     "铁喷",
		"w_shotgun_m1014":    "铁喷",
		"autoshotgun":        "一代连喷",
		"w_autoshot_m4super": "一代连喷",
		"shotgun_spas":       "二代连喷",

		// 冲锋枪
		"smg_uzi":      "乌兹",
		"w_smg_uzi":    "乌兹",
		"smg_a":        "消音",
		"smg_silenced": "消音",
		"w_smg_mp5":    "MP5",
		"smg_mp5":      "MP5",

		// 手枪
		"pistol_magnum":  "马格南",
		"w_desert_eagle": "马格南",
		"pistol_glock":   "小手枪",
		"w_pistol_glock": "小手枪",
		"w_pistol_b":     "小手枪",

		// 投掷物
		"grenade_launcher": "榴弹",

		// 近战武器
		"melee_machete":   "砍刀",
		"w_machete":       "砍刀",
		"machete":         "砍刀",
		"melee_katana":    "武士刀",
		"w_katana":        "武士刀",
		"katana":          "武士刀",
		"baseball_bat":    "棒球棍",
		"w_bat":           "棒球棍",
		"melee_knife":     "匕首",
		"w_knife_t":       "匕首",
		"chainsaw":        "电锯",
		"w_chainsaw":      "电锯",
		"crowbar":         "撬棍",
		"w_crowbar":       "撬棍",
		"fireaxe":         "消防斧",
		"w_fireaxe":       "消防斧",
		"frying_pan":      "平底锅",
		"w_frying_pan":    "平底锅",
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

	// CS系列武器检测（第二优先级）
	csWeapons := map[string]string{
		"ak47":         "ak47",
		"ak74":         "ak74",
		"awp":          "AWP",
		"glock":        "glock",
		"m4a1":         "M4A1",
		"m4a4":         "M4A4",
		"deagle":       "deagle",
		"desert_eagle": "deagle",
		"usp":          "USP",
		"p90":          "P90",
		"mp5":          "MP5",
		"famas":        "FAMAS",
		"galil":        "galil",
		"aug":          "AUG",
		"scout":        "scout",
		"m249":         "M249",
		"xm1014":       "XM1014",
		"m3":           "M3",
		"tmp":          "TMP",
		"mac10":        "MAC10",
		"ump45":        "UMP45",
	}

	// 检测CS武器
	for keyword, weaponCode := range csWeapons {
		if strings.Contains(lowerFilename, keyword) {
			secondaryTags[weaponCode] = true
			return
		}
	}
}
