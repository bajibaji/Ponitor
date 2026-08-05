# RETRO_GUI 交接文档

> 本文件是项目的**交接文档**（替代旧 handoff）。任何会话结束/交接时，更新本文件；新会话开始时先读本文件对齐现状。

最后更新：2026-08-02（v0.5.1 发布后）

## 项目是什么

**Ponitor** —— Windows 托盘系统监控，Go 单二进制。内嵌 CRT 像素风 HTTP 仪表盘，把旧手机变成 PC 的实时性能监视副屏。

- 后端：Go + `systray` + `gopsutil` + PDH/WMI GPU
- 前端：`dashboard.html` 纯静态（`//go:embed` 嵌入二进制），**不做本地化**（用户明确要求）
- 运行：零依赖单二进制，默认 2s 采集，内存 ~20MB

## 当前版本

- **v0.5.1**（2026-08-01 发布，tag `v0.5.1`）
- Release 产物：`Ponitor_<版本>_windows_x64.exe` / `windows_arm.exe`（CI 自动构建，命名 x64/arm）
- 版本号注入：`-ldflags "-X main.version=..."`（rebuild.bat 与 CI 均已接入）

## 功能盘点（v0.5.1）

| 模块 | 说明 |
|---|---|
| 监控项 | CPU（gopsutil）/ 内存 / GPU / 网络速率 |
| GPU | NVIDIA 走 `nvidia-smi`（利用率/显存/温度）；AMD/Intel 走 PDH 计数器 + WMI（利用率+名称，显存温度 N/A） |
| 托盘菜单 | 打开网页 / 扫描间隔 / 主题配色 / 卡片高度 / 显示用户名 / 界面语言 / 开机自启 / 作者+版本（灰色）/ 退出 |
| 卡片高度 | **自适应 (Auto, 默认)** / 标准 180 / 适中 150 / 紧凑 110 / +10 / -10；Auto 按 `window.innerHeight` 实时算：横屏 (vh−54)/2、竖屏 (vh−92.6)/4 |
| 进度条 | 点线+色块覆盖式；点数随容器宽度动态生成（实测点宽），窄屏不溢出 |
| 语言 | 托盘菜单单一语言，默认跟随系统（`LANG` 环境变量 zh*→中文），可切 中/英，持久化 `config.lang` |
| 用户名 | 仪表盘 `root@<用户名>`，默认系统账户名，托盘可自定义（/setname 页） |
| 开机自启 | 注册表 `HKCU\...\Run` 键名 `Ponitor`（auto_start_windows.go） |
| 单实例 | 全局互斥体 `Global\Ponitor`，重复启动直接退出 |

## 菜单结构（v0.5.1）

```
打开网页 / Open Web
──────────
扫描间隔 / Interval（0.5/1/2/5 秒）
主题 / Theme（黑绿 Matrix / 琥珀金 Amber / 赛博蓝 Cyber Blue / 黑白 Classic Mono）
卡片高度 / Card Height（自适应 Auto / 标准 180 / 适中 150 / 紧凑 110 ── +10 / -10）
显示用户名 / Username（自定义… / 恢复系统用户名）
界面语言 / Language（自动 / 中文 / English）
开机自启 / Run at Startup
──────────
作者：DanJuan（灰色只读）
v0.5.1（灰色只读，构建注入）
──────────
退出程序 / Quit
```

## 关键文件

| 文件 | 职责 |
|---|---|
| `main.go` | 采集 + HTTP API + 托盘菜单 + 配置 |
| `dashboard.html` | 前端仪表盘（嵌入二进制） |
| `gpu_windows.go` / `gpu_other.go` | GPU 采集（PDH+WMI / stub） |
| `hide_windows.go` / `hide_other.go` | 隐藏控制台 + 开浏览器 + 单实例互斥体 |
| `auto_start_windows.go` / `auto_start_other.go` | 开机自启注册表 |
| `lanip.go` | 局域网 IP |
| `Cubic_11.woff2` + `OFL.txt` | 像素字体（SIL OFL 1.1，随二进制分发） |
| `.github/workflows/build.yml` | Windows x64/arm CI 构建，tag v* 触发 |

## 构建与发布

```bash
# 本地构建（rebuild.bat 自动取 git tag 命名 + 注入版本）
.\rebuild.bat

# 发布（CI 自动构建 + Release 附产物）
git tag v0.5.2 && git push origin v0.5.2
```

- Release 更新日志需**中英双语**手写（`gh release edit <tag> --notes-file ...`）
- 发布后更新：README（版本号/功能）、本文件「当前版本」

## API

`/api/cpu` `/api/mem` `/api/gpu` `/api/network` `/api/config` `/api/theme` `/api/interval` `/api/cardheight`（h=0 自适应）/ `/api/username` / `/setname`（用户名设置页）/ `/font/Cubic_11.woff2`

## 已知约束 / 用户偏好

- **dashboard.html 不本地化**（托盘菜单才做语言切换）
- 托盘中文菜单**顶级 4 字对齐**（用户审美偏好：扫描间隔/主题配色/卡片高度/显示用户名/界面语言/开机自启）
- 进度条：用户要求 `█` 填充 + 连续点线，`]` 位置必须恒定（多轮迭代的坑：字符宽度方案全失败，最终用「点线底 + 绝对定位色块覆盖」）
- 卡片高度固定开销实测：横屏 54px（iPhone 7P 414 视口 − 2×180 反推）、竖屏 92.6px
- iPhone 7P 是主要测试机（横屏），iPhone 4 老手机也验证过（Safari 上滑隐藏地址栏）
- 截图只保留横屏两张（screenshot.png 横屏 / screenshot3.png 老手机）

## 待办 / 已知问题

- （无已知未决问题；新发现写这里）

## 交接检查清单

- [ ] 版本号与 Release 对齐（main.go 注入、README、本文件）
- [ ] 未提交改动已 commit/push
- [ ] 功能盘点反映当前菜单实际结构
