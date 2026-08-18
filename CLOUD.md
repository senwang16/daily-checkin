# 进阶：GitHub Actions 云端模式（不开本机也能每天签到）

把签到放到 GitHub 免费定时任务上跑，**不依赖本机开机**，每天北京时间 09:30 自动执行。
本地 exe 仍照常工作；云端是「双保险 / 移动场景」的额外通道。

> 状态：可用。仓库已内置 `.github/workflows/daily-checkin.yml`，按本文设置即可。

---

## ⚠️ 风险提示（务必先看）

- **WorkBuddy 云端：安全可用。** 只需 `WORKBUDDY_TOKEN`（桌面端明文 token），服务端只校验 token 有效，**不校验来源 IP/机器**。
- **Trae 云端：尽力而为，可能触发 9074 风控。** Trae 的 `claim` 接口除校验 token 外，还会校验**设备指纹**。
  把本机提取到的真实 aha 设备 ID 作为 `TRAE_AHA_ID` 传入，比「随机 device」稳得多，
  但云端请求来自 GitHub 服务器 IP，Trae 仍可能判定为异常登录而返回 `9074 操作太过频繁`。
  - 若只想 100% 稳，云端只勾选 `workbuddy` 即可（Trae 留本机 exe 跑）。
  - 若云端 Trae 被 9074，按下方「续期/重提取」重新导出 Secrets 即可，不会封号。
- **token 续期**：Trae `refresh_token` 约 180 天、每天跑会被自动续期并回写本机；
  但**云端读取的是 Secrets 里的快照**，不会自动更新。WorkBuddy token 也会随桌面端过期。
  所以建议：**每隔一两个月重新 `export` 一次、更新 Secrets**（或本机 exe 跑着顺带续，但云端不共享）。
- **Secrets 安全**：任何有仓库**写入权限**的人都能读到你的 token。
  本仓库**源码可公开**（里面不含任何 token/secret，设备指纹也是运行时从本机提取）；
  但如果你用云端模式**跑自己的 token**，该运行仓库建议设为 **Private（私有）**，避免协作者看到。

---

## 第一步：本机提取凭据（写入 Secrets 的原料）

在本机（已登录 Trae / WorkBuddy 桌面端）执行：

```bash
daily-checkin.exe export
```

终端会打印形如：

```
WORKBUDDY_TOKEN=eyJhbGci...xxxxxxxx
TRAE_AHA_ID=<你的aha设备ID，例如 1234567890123456>
TRAE_AUTH_JSON={
  "access_token": "xxx",
  "refresh_token": "xxx",
  "uid": "xxx",
  "device_id": "<你的aha设备ID>",
  ...
}
CHECKIN_PLATFORMS=trae,workbuddy
```

把这些值复制下来，下一步用。

> 原理：工具从本机读取
> - WorkBuddy token：`%LOCALAPPDATA%\CodeBuddyExtension\Data\Public\auth\workbuddy-desktop.info`
> - Trae 凭据：`%LOCALAPPDATA%\daily-checkin\trae_auth.json`（或旧 `checkin/auth.json`）
> - aha 设备 ID：`%APPDATA%\TRAE SOLO CN\logs\aha_electron_*.log`
>
> 云端模式下，程序**优先读环境变量**（`WORKBUDDY_TOKEN` / `TRAE_AUTH_JSON` / `TRAE_AHA_ID`），不再依赖本机文件。

## 第二步：在 GitHub 创建仓库并写入 Secrets

1. 打开 https://github.com/new
   - Repository name：`daily-checkin`
   - **只跑云端 token 的话建议选 Private（私有）**；若仅分享源码、不存 secret，Public 也可
   - 不用勾 Initialize（我们本地推）
2. 进入仓库 `Settings → Secrets and variables → Actions → New repository secret`，逐个新建：

   | Name | 值（来自第一步） |
   |---|---|
   | `WORKBUDDY_TOKEN` | `export` 里的 `WORKBUDDY_TOKEN=` 后面整串 |
   | `TRAE_AUTH_JSON` | `export` 里的整个 JSON 块（可多行） |
   | `TRAE_AHA_ID` | `export` 里的 `TRAE_AHA_ID=` 后面数字 |
   | `CHECKIN_PLATFORMS` | `trae,workbuddy`（只想要 WorkBuddy 就写 `workbuddy`） |

## 第三步：把仓库推到 GitHub

在 `daily-checkin/` 目录：

```bash
git init
git add .
git commit -m "init: daily-checkin (windows exe + cloud workflow)"
git branch -M main
git remote add origin https://github.com/<你的用户名>/daily-checkin.git
git push -u origin main
```

> 本地构建产物 `daily-checkin.exe` 已在 `.gitignore` 忽略，仓库只含源码，CI 每次自己 `go build`。

## 第四步：启用并验证

1. 仓库 `Actions` 页 → 左侧 `daily-checkin` → `Enable workflow`（若提示）。
2. 点 `Run workflow` 手动触发一次，看日志是否两个平台都 `[OK]`。
3. 之后每天 UTC 01:30（北京 09:30）自动跑；失败会自动开 Issue 提醒。

---

## 续期 / 重提取（token 过期或被风控时）

1. 本机重新 `daily-checkin.exe export`
2. 回到 GitHub 仓库 Secrets，更新 `WORKBUDDY_TOKEN` / `TRAE_AUTH_JSON` / `TRAE_AHA_ID`
3. 手动 `Run workflow` 验证

---

## 与本地 exe 的关系

- 两者**共用同一套 Go 代码**，只是触发方式不同：本机走计划任务/登录触发，云端走 Actions。
- 同一天两个通道都跑也没事：接口幂等，**今日已签不会重复计、也不会报错**。
- 想纯云端、本机不装 → 不运行本地 exe 的安装向导即可；想要双保险 → 两边都开着。
