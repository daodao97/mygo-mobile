<p align="center">
<img width="800" height="360" alt="MyGo!!!!!" src="https://github.com/user-attachments/assets/c3eee9bc-5f22-4032-95b6-776e8381910a" />
</p>

**Desktop apps in Go, with native UI or a web frontend.**

> [!NOTE]
> 这是 [egoist/mygo](https://github.com/egoist/mygo) 的 fork（`daodao97/mygo`），在上游之上增加了 **iOS / Android 原生 UI** 和一批桌面扩展。
> 与上游的差异、合并规则和进行中的 PR 见 [FORK.md](FORK.md)。

## 关于本 fork

### 为什么有这个 fork

上游只面向桌面（macOS、Linux、Windows）和浏览器。本 fork 把同一套 Go 原生 UI 跑到手机上，让一个应用的界面和逻辑在桌面与移动端同构，例如终端工具 [Retty](https://github.com/daodao97/retty)（macOS + iOS）和 Miya。

移动端曾提交给上游，被维护者婉拒：

| PR | 内容 | 结果 |
|---|---|---|
| [egoist/mygo#157](https://github.com/egoist/mygo/pull/157) | Add experimental iOS native UI and mobile build tooling：UIKit 后端、触摸/手势/输入法、移动端系统能力、APNs、`mygo build -platform ios`、Xcode 工程生成与签名导出（206 个文件，+19,699 行） | 2026-10-08 关闭，未合并。维护者的答复：*“I don't want to maintain ios/android support, and I also prefer native ui components on mobile device.”* |

因此移动端代码**只在本 fork 维护**，不再向上游提交。Android 后端在
[@Heyana](https://github.com/Heyana) 贡献的实现基础上合入并继续补齐。

### 比上游多出的功能

**移动端（仅 fork）**

- **iOS**：UIKit 宿主 + Go 视图，CoreText 排版、Metal 渲染（CPU 回退），安全区、键盘、显示链路帧调度（支持 ProMotion 120 Hz）、场景生命周期与状态恢复；`mygo build -platform ios/arm64`、模拟器构建、签名、IPA 导出与 dSYM。见 [docs/ios.md](docs/ios.md)。
- **Android**：JNI 后端与 Gradle 宿主工程，Surface/EGL 恢复、生命周期与深链、原生编辑菜单与选区手柄、HarfBuzz 文字排版；构建脚本见 `scripts/android-*.sh`。
- **共享 UI 的触摸支持**：触摸滚动与弹性滚动、捏合/旋转手势、交互式路由返回、输入法与输入附件栏、原生选区/光标几何。
- **移动端系统能力**：推送（APNs、Live Activity 及 `push/` 发送端）、生物识别、权限、相机扫码、分享、语音与麦克风 PCM 流、App Group、后台刷新/后台任务、触觉、Keychain/Keystore 上的 `Secrets`。
- 配套包：`sharedstore`、`generationcache`、`async`、`backgroundtask`、`timeseries`、`ui/chart/sparkline`。

**对上游的通用扩展（桌面也可用）**

- `plugins/desktopcontrol`：macOS 显示器/窗口枚举、ScreenCaptureKit 截图、键鼠输入、辅助功能对话框（纯 Go）。
- 网络路径状态 `Network.PathStatus` / `OnPathChanged`。
- 路由 `Router.Reset` / `History` / `Transitioning` 与 JSON 持久化。
- inline combobox、`DecodeBitmapSize`（按显示尺寸解码图片）、`LastInput`。
- 应用状态持久化 `PersistState`、`OnFirstFrame`、`ui.Startup`。
- 诊断：`FrameReading` / `FrameAverage` / `HeapReadingNow`、`MYGO_FRAME_WHY` 帧原因追踪、`LoopFPS`、`internal/probe` 性能探针。
- 终端插件：`AdaptiveColors`、`SelectOnDrag`、`SelectedText` / `SelectAll` / `CopyRawText`、远程固定网格（`FixedCols`、`ReadScreen` 等）、TUI 硬换行路径识别。
- `updater.InstallUpdate`、`automation.Start`。

完整清单与代码位置见 [FORK.md](FORK.md) 的「差异分类」。

### 与上游的同步策略

- **同一功能两套实现时，以上游为准。** 上游落地了 fork 也有的功能，就删掉 fork 的版本，把调用方（Miya、myclash 及 fork 自身）迁到上游 API，不因为 fork 先做了而保留。上游没有的功能继续保留。
- **定期合并**：`git fetch upstream && git merge --no-ff upstream/main`，冲突按上一条处理；合并后编译桌面、iOS、Android 与下游应用，再快进推送 `main`，从不强推。上游给 `platform` 接口新增的方法要在 iOS/Android 后端补上。
- **合并后人工排查**：git 不会提示“同一功能、无文本冲突”的重叠，需要对照上游本次改动找出来按规则处理，并更新 FORK.md。
- **回馈上游**：通用的修复和功能从 `upstream/main` 上的独立 worktree 提取，剥离移动端代码，以草稿 PR 提交。
- 桌面构建始终保持 `CGO_ENABLED=0`；cgo 只用于移动端。

### 提交到上游的 PR

| PR | 内容 | 状态 |
|---|---|---|
| [#157](https://github.com/egoist/mygo/pull/157) | iOS 原生 UI 与移动端构建工具链 | 已关闭（上游不维护移动端） |
| [#928](https://github.com/egoist/mygo/pull/928) | macOS 空闲窗口不再唤醒 Go | 已合并 |
| [#1057](https://github.com/egoist/mygo/pull/1057) | Markdown 中 CJK 旁的 `*` / `~` 强调 | 已合并 |
| [#1058](https://github.com/egoist/mygo/pull/1058) | 链接悬停下划线完整穿过下行字母 | 已合并 |
| [#1076](https://github.com/egoist/mygo/pull/1076) | 终端可选的 zsh shell integration | 关闭，上游在此基础上另行实现 |
| [#1301](https://github.com/egoist/mygo/pull/1301) | 窗口空闲时释放文本 shaping 缓冲 | 关闭，上游已有等价的 `System.ReleaseShaping` |
| [#1302](https://github.com/egoist/mygo/pull/1302) | 终端识别中文文本中的路径、带引号路径与行列定位 | 已合并 |
| [#1305](https://github.com/egoist/mygo/pull/1305) | 终端同步更新 1 秒后放行 | 已合并 |
| [#1525](https://github.com/egoist/mygo/pull/1525) | `plugins/desktopcontrol` 官方插件 | 草稿，进行中 |
| [#1530](https://github.com/egoist/mygo/pull/1530) | macOS 原生 popover 跟随应用外观 | 草稿，进行中 |

### 使用本 fork

模块路径仍是 `github.com/egoist/mygo`，在应用的 `go.mod` 里用 `replace` 指向本 fork：

```
replace github.com/egoist/mygo => github.com/daodao97/mygo <version>
```

以下是上游的原始介绍。

---

Sponsored by:

<p align="center">
  <a href="https://openuptime.app/?utm_source=github&utm_medium=sponsor&utm_campaign=mygo">
    <picture>
      <source media="(prefers-color-scheme: dark)" srcset="https://openuptime.app/sponsor/banner-dark.png">
      <img width="600" alt="OpenUptime: open-source uptime monitoring and status pages" src="https://openuptime.app/sponsor/banner-light.png">
    </picture>
  </a>
</p>

---

Each MyGo window shows one of two kinds of interface, and one app, or one
window, can mix them:

- **A web page**, in the webview the OS already has: WKWebView on macOS,
  WebKitGTK on Linux, WebView2 on Windows. Build the frontend with any web
  tools; it calls your Go code through a TypeScript client generated from it.
- **Native UI**, written in Go alone with package `ui` and
  drawn by MyGo itself on the GPU: no HTML, no JavaScript and no webview, so
  the window opens at once and takes little memory.

Either way, an app is a single Go binary of a few megabytes, focused on low
memory and CPU use.

- **Desktop without cgo**: cross-compile macOS, Linux and Windows from any machine.
- **Experimental iOS native UI**: a UIKit host with Go views, CoreText and
  Metal; build with cgo on macOS and Xcode. See [iOS](docs/ios.md).
- **Typed IPC** for pages: bind Go services, stream values through channels
  and send typed events; the TypeScript client is generated from your Go
  code.
- **A Go UI toolkit** for native UI: flexbox and grid layout, widgets, text
  editing with input methods, virtualized lists, SVG icons, animations,
  screen reader support, and views you test without a window; Liquid Glass
  on every platform with the glass plugin; web pages inside native UI, under
  the menus and dialogs it shows over them.
- **Desktop APIs** for both: windows, menus, tray, dialogs, notifications,
  global shortcuts, deep links, file associations, recent documents, Now
  Playing and the media keys, the system accent color and accessibility
  settings, watching files, and more.
- **Ready to ship**: app bundles and disk images, Windows installers, Debian
  packages, PKGBUILDs for Arch Linux and a Linux install script, code signing, notarization, and signed
  auto-updates with delta updates and an update window in the manner of
  Sparkle.

A window with a web page, whose frontend calls Go:

```go
type Greeter struct{}

// Greet returns a greeting.
func (Greeter) Greet(name string) string { return "Hello, " + name }

func main() {
	mygo.Bind(Greeter{})
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{Title: "Hello", URL: "/"})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
```

```ts
import { Greeter } from "./mygo"; // generated

document.body.textContent = await Greeter.greet("Ada");
```

A window with native UI, whose view is a Go function of the app's state:

```go
type counter struct{ n int }

func (s *counter) view(c *ui.Context) {
	ui.Column(c).Fill().Center().Gap(12).Children(func() {
		ui.Text(c, fmt.Sprint(s.n)).FontSize(40).Bold()
		if ui.PrimaryButton(c, "Increment").Clicked() {
			s.n++
		}
	})
}

func main() {
	s := &counter{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Counter",
			Content: ui.View(s.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
```

## Getting started

With [Go](https://go.dev/dl/) 1.27.3+ and [Bun](https://bun.sh), create an app
with a web frontend:

```sh
bunx mygo-cli init my-app      # or: npx mygo-cli init my-app
cd my-app
bun run dev
```

An app of native UI needs Go alone:

```sh
go run github.com/egoist/mygo/cmd/mygo@latest init -template native my-app
cd my-app
go tool mygo dev
```

Read the [documentation](docs/README.md).

## License

MIT
