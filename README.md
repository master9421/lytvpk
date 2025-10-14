# LytVPK

一个专为 Left 4 Dead 2 (L4D2) 设计的现代化 VPK 插件管理工具。
> 该项目通篇使用AI生成，本人只在AI陷入困境时进行少量修改。仔细阅读代码你就会发现大量无用变量、不符合规范的函数定义，一个文件几千行的屎山，均不代表本人水平，谢谢！

![LytVPK](https://img.shields.io/badge/Platform-Windows-blue)
![Build](https://img.shields.io/badge/Build-Wails_v2-green)
![Language](https://img.shields.io/badge/Language-Go_+_JavaScript-orange)

## 🚀 功能特性

### 核心功能
- **智能扫描**: 自动扫描和解析 VPK 文件，提取详细的内容信息
- **内容识别**: 智能识别地图、武器、角色、音频等游戏内容类型
- **标签系统**: 自动生成标签，支持按类型、位置、内容筛选
- **批量管理**: 支持批量启用/禁用 VPK 文件

## 🎮 支持的内容类型

### 地图和模式
- 战役模式 (Campaign)
- 对抗模式 (Versus) 
- 生存模式 (Survival)
- 清道夫模式 (Scavenge)
- 突变模式 (Mutation)
- 写实模式 (Realism)

### 武器类型
- 步枪、突击步枪、狙击枪
- 霰弹枪、手枪、冲锋枪
- 近战武器：匕首、斧头、电锯、砍刀等

### 角色模型  
- **幸存者**: Bill, Francis, Louis, Zoey, Coach, Ellis, Nick, Rochelle
- **感染者**: Boomer, Hunter, Smoker, Witch, Tank, Charger, Jockey, Spitter

### 其他内容
- 音频文件（音乐、语音、音效、环境音）
- 材质文件（皮肤、界面、粒子效果）
- 脚本文件（Squirrel脚本、配置文件）
- UI界面和特效文件

## 🛠️ 技术架构

### 后端 (Go)
- **框架**: Wails v2
- **VPK解析**: 使用 `git.lubar.me/ben/valve/vpk` 库
- **并发处理**: `github.com/panjf2000/ants/v2` 协程池
- **配置管理**: JSON 格式的持久化配置

### 前端 (JavaScript + CSS)
- **原生 JavaScript**: 无框架依赖，轻量高效
- **现代 CSS**: 基于 CSS 变量的设计系统
- **响应式设计**: 支持桌面端和移动端
- **实时通信**: 通过 Wails 事件系统与后端通信

## 📦 安装和使用

### 系统要求
- Windows 10/11

### 使用说明
1. **选择目录**: 点击"选择L4D2目录"按钮，选择游戏的 addons 文件夹
2. **扫描文件**: 应用会自动扫描并解析所有 VPK 文件
3. **管理插件**: 使用界面上的开关来启用/禁用插件
4. **筛选搜索**: 使用搜索框和标签筛选来查找特定插件
5. **批量操作**: 选择多个文件进行批量启用/禁用

## 🙏 致谢

- [Wails](https://wails.io/) - 跨平台应用框架
- [valve/vpk](https://git.lubar.me/ben/valve) - VPK 文件解析库
- [ants](https://github.com/panjf2000/ants) - 高性能协程池
