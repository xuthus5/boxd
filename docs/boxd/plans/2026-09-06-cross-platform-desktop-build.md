# 跨平台桌面应用构建能力补齐计划

## 背景

当前 boxui 项目的桌面应用打包脚本存在以下缺失：
1. **Windows**：仅生成 zip 压缩包，未集成 NSIS 安装程序（无开始菜单、卸载程序、WebView2 引导等）
2. **Linux**：已有 deb/rpm/AppImage，但缺少 Flatpak 支持
3. **macOS**：已有 .app bundle 和 DMG，基本完整

参考 mssh 项目的成熟实践，需要补齐 Windows NSIS 集成，并评估其他平台的增强需求。

## 当前状态分析

### 现有脚本
- `scripts/package-desktop.sh`：Linux 打包（deb/rpm/AppImage）
- `scripts/package-desktop-windows.sh`：Windows 仅构建 exe + zip
- `scripts/package-desktop-macos.sh`：macOS 打包（.app + zip + DMG）

### 缺失能力（Windows）
1. NSIS 安装程序生成
2. WebView2 运行时引导程序集成
3. 开始菜单/桌面快捷方式创建
4. 卸载程序与安装日志
5. 版本信息嵌入（info.json）
6. 代码签名支持（可选）

## 实施计划

### 第一阶段：Windows NSIS 集成

#### 步骤 1：创建 NSIS 配置目录结构
```
desktop/build/windows/nsis/
├── project.nsi          # NSIS 主安装脚本
├── wails_tools.nsh      # Wails 工具函数（参考 mssh）
└── info.json            # 版本信息（可选）
```

#### 步骤 2：适配 NSIS 脚本
- 从 mssh 项目复制 `project.nsi` 和 `wails_tools.nsh`
- 修改应用名称、图标、版本信息等变量
- 确保与 boxd 的目录结构兼容（数据目录、配置路径）

#### 步骤 3：修改 Windows 打包脚本
更新 `scripts/package-desktop-windows.sh`：
1. 添加 NSIS 安装程序生成步骤
2. 集成 WebView2 引导程序（`wails3 generate webview2bootstrapper`）
3. 生成版本信息文件（可选）
4. 保持向后兼容（仍生成 zip）

#### 步骤 4：添加依赖检查
- 检测 `makensis` 是否安装
- 在 CI 环境中安装 NSIS（Windows runner 可通过 Chocolatey 安装）
- 提供友好的错误提示

### 第二阶段：Linux Flatpak 支持（可选）

#### 步骤 5：创建 Flatpak 配置
- 创建 `desktop/build/linux/flatpak/` 目录
- 编写 `com.boxd.desktop.yml` 清单文件
- 配置运行时、SDK 和权限

#### 步骤 6：修改 Linux 打包脚本
更新 `scripts/package-desktop.sh`：
1. 添加 Flatpak 构建步骤
2. 生成 Flatpak bundle（`.flatpak`）

### 第三阶段：CI 集成

#### 步骤 7：更新 GitHub Actions 工作流
- 在 `release.yml` 中添加 Windows NSIS 安装程序构建
- 在 `nightly.yml` 中添加 Windows 安装程序构建
- 上传安装程序作为发布资产

#### 步骤 8：更新 Makefile
添加 `package-desktop:nsis` 和 `package-desktop:flatpak` 目标。

## 技术细节

### NSIS 关键配置
```nsis
; 应用信息
!define APP_NAME "boxd-desktop"
!define APP_VERSION "${VERSION}"
!define APP_PUBLISHER "boxd developers"
!define APP_WEB_SITE "https://github.com/xuthus5/boxd"

; 安装目录
InstallDir "$PROGRAMFILES\${APP_NAME}"
InstallDirRegKey HKLM "Software\${APP_NAME}" "InstallDir"

; 组件定义
Section "MainSection" SEC01
  ; 安装主程序
  SetOutPath "$INSTDIR"
  File "bin\boxd-desktop.exe"
  ; 创建卸载程序
  WriteUninstaller "$INSTDIR\uninstall.exe"
  ; 写入注册表
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_NAME}" \
    "DisplayName" "${APP_NAME}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_NAME}" \
    "UninstallString" '"$INSTDIR\uninstall.exe"'
SectionEnd
```

### WebView2 引导
使用 wails3 生成 WebView2 引导程序：
```bash
wails3 generate webview2bootstrapper -dir "desktop/build/windows/nsis"
```

### 版本信息嵌入
通过 `-ldflags` 注入版本：
```bash
-ldflags "-X github.com/xuthus5/boxd/internal/core.Version=${VERSION}"
```

## 验证方案

### 本地验证
1. 在 Windows 机器上安装 NSIS
2. 运行 `./scripts/package-desktop-windows.sh 0.1.0 amd64`
3. 验证生成的安装程序：
   - 安装/卸载流程
   - 开始菜单快捷方式
   - WebView2 引导（如未安装）
   - 版本信息显示

### CI 验证
1. 推送分支并触发 CI
2. 检查 Windows 构建日志
3. 下载生成的安装程序进行测试

## 风险与缓解

1. **NSIS 安装包体积增大**
   - 缓解：使用 LZMA 压缩，排除调试符号

2. **WebView2 引导失败**
   - 缓解：提供离线安装包作为备选

3. **代码签名缺失**
   - 缓解：文档说明如何添加签名，暂不实施

4. **Flatpak 权限配置**
   - 缓解：参考 mssh 项目的 Flatpak 配置

## 时间估算

- Windows NSIS 集成：2-3 小时
- Linux Flatpak 支持：1-2 小时
- CI 集成：1 小时
- 测试与验证：1 小时

总计：5-7 小时

## 依赖项

1. NSIS 工具（`makensis`）
2. wails3 CLI（已存在）
3. WebView2 引导程序（wails3 生成）
4. Flatpak 构建工具（flatpak-builder，可选）

## 后续步骤

1. 用户确认计划范围
2. 实施 Windows NSIS 集成
3. 测试安装程序生成
4. 更新 CI 工作流
5. 文档更新

## 参考资源

- mssh 项目 NSIS 配置：`/root/projects/mssh/build/windows/nsis/`
- wails3 NSIS 文档：https://wails.io/docs/guides/distribution
- NSIS 官方文档：https://nsis.sourceforge.io/