# daily-checkin

Windows 下 **Trae** 与 **WorkBuddy** 的每日自动签到工具。单文件 exe，双击即用，不依赖 Python 运行时，不依赖任何 GUI 库，普通 Windows 命令行即可运行。

- **Trae**：每日 +200 积分
- **WorkBuddy**：每日 +100 积分
- 注册为 Windows 计划任务，每天定时自动跑；计划任务被禁用时自动降级为开机自启常驻
- 签到失败弹 **Windows 通知中心 Toast**，并写明具体原因（token 失效 / 风控 / 网络等）

---

## 特性

- 单 exe，零依赖，小白双击即用
- 安装向导：双击后进入命令行交互，自动检测本机已登录平台，选平台 + 设时间
- 失败重试（指数退避）+ 明确错误分类码与中文说明
- 全程不联网上传任何数据，凭据仅存于本机

## 支持的签到平台

| 平台 | 每日 | 凭据来源 |
|---|---|---|
| Trae | +200 | 本机 `trae_auth.json` + Trae App 真实设备指纹（aha 设备 ID，从 App 日志提取） |
| WorkBuddy | +100 | 本机 `workbuddy-desktop.info` 明文登录态 |

> **Trae 说明**：v1 从本机已有 `trae_auth.json` 读取 token（兼容旧 checkin 项目的 `auth.json`）。
> 后续版本会内置 Trae OAuth 登录向导自动生成该文件。当前若没有，请把旧项目的 `auth.json`
> 放到 `%LOCALAPPDATA%\daily-checkin\trae_auth.json`。

## 快速开始（小白）

1. 下载 `daily-checkin.exe`（单文件，零依赖）
2. **双击**打开安装向导（会弹出一个黑色命令行窗口，这是正常的）
3. 按提示输入每天签到时间（回车默认 `09:30`）
4. 按 `Y/N` 选择是否启用 Trae / WorkBuddy（已自动检测本机登录态）
5. 回车确认安装。此后每天到点自动签到，失败会弹通知

> **提示**
> - 双击 `daily-checkin.exe` 就是安装/配置向导，直接在命令行里问答，既能首次安装也能改时间/平台。
> - 也可以用同目录的 **`配置.bat`** 一键启动同一个向导（修改配置时双击它或 exe 都行，效果一样）。
> - 完全不想交互、一键用默认（两平台 / 每天 09:30）安装：打开 `cmd` 执行 `daily-checkin.exe install`。
> - 若 Windows 弹出 SmartScreen 安全提示（未签名程序的正常提示），点「更多信息」→「仍要运行」即可。

## 命令行

```
daily-checkin.exe            # 打开交互式安装向导（默认，双击即用）
daily-checkin.exe run        # 立即执行一次签到（计划任务每天调用）
daily-checkin.exe run --daemon   # 常驻定时器模式（降级自启时使用）
daily-checkin.exe status     # 查看安装与登录状态
daily-checkin.exe install    # 静默安装（默认两平台 / 09:30）
daily-checkin.exe install --platforms=trae,workbuddy --time=09:30
daily-checkin.exe uninstall  # 卸载（清理计划任务/自启与配置）
```

### 静默安装（一键默认）

如果你已经知道要装什么，直接一条命令搞定：

```
daily-checkin.exe install --platforms=trae,workbuddy --time=09:30
```

安装完成后会注册两个 Windows 计划任务（见下文「开机自启」）。

## 故障排查

**安装后任务计划程序里只看到 `DailyCheckin`，没有 `DailyCheckin-Logon`？**
- 这是旧版安装或登录触发任务注册失败的迹象。新版安装应同时创建两个任务。
- 修复：用新版 exe 重新运行安装向导（双击 `daily-checkin.exe` 或 `配置.bat`），如果提示「登录触发任务注册失败」请把错误信息发给我。
- 手动补齐登录触发任务（以管理员身份运行 cmd，复制粘贴）：
  ```bat
  schtasks /Create /TN "DailyCheckin-Logon" /TR "\"C:\path\to\daily-checkin\daily-checkin.exe\" run" /SC ONLOGON /RL HIGHEST /F
  ```
  然后刷新任务计划程序，确认出现 `DailyCheckin-Logon`。

**双击 exe 被 SmartScreen 拦截？**
- 点「更多信息」→「仍要运行」即可。本程序未签名，属正常提示。

**双击 exe 黑框一闪而过？**
- 正常现象：本程序默认是命令行交互向导，双击后会**停留**在黑框里等你输入时间/平台。
- 如果确实闪退，查看日志 `%LOCALAPPDATA%\daily-checkin\checkin.log` 或 `%LOCALAPPDATA%\daily-checkin\gui_error.log`。
- 备用方案：双击 `配置.bat`，效果相同。

## 错误码

| 码 | 含义 | 处理建议 |
|---|---|---|
| E001 | 未找到登录凭据 | 先在本机登录该平台，重新运行安装 |
| E002 | token 失效 | 重新运行安装以刷新凭据 |
| E003 | 风控拦截(9074) | 设备指纹不匹配，重新运行以提取本机设备信息 |
| E004 | 网络错误 | 检查网络后重试 |
| E005 | 业务未知错误 | 看通知里的服务端 message |
| E006 | 计划任务注册失败 | 手动检查任务计划程序 |

## 原理

- **WorkBuddy**：直接调用官方签到 API `POST https://www.codebuddy.cn/v2/billing/meter/daily-checkin`，
  读取桌面端明文登录态 token，与积分余额无关，0 积分也能签到。
- **Trae 绕过 9074 风控**：Trae 的 `claim` 接口会校验**设备指纹**，必须是本机 Trae App 注册的真实
  aha 设备 ID。工具从 `%APPDATA%\TRAE SOLO CN\logs\aha_electron_*.log` 提取该 ID 并带入请求头，
  从而通过风控（普通随机 device 会被 `9074 操作太过频繁` 拦截）。

## 扩展新平台

实现 `platform.Platform` 接口（`Name` / `DisplayName` / `Detect` / `Checkin`），在 `platform/init()` 注册即可。
新增平台完全独立，不影响其他逻辑。

## 安全

- 所有 token 仅存储于本机（`%LOCALAPPDATA%\daily-checkin\`），不会上传到任何服务器
- 开源仓库不含任何个人凭据

## 编译

需要 Go 1.22+（首次 `go build` 会自动拉取匹配的工具链）：

```
go build -o daily-checkin.exe .
```

## 许可证

MIT — 见 [LICENSE](LICENSE)
