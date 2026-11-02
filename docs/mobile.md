# Phones and tablets

The same MyGo app runs on iPhones and iPads, and on Android phones and
tablets: windows of web pages show in the system's web view (WKWebView on
iOS, Android's WebView), windows of native UI draw with the GPU (Metal on
iOS, OpenGL ES on Android), and the app's Go code, its bound services and
its events are the same. `mygo dev` runs the app on a simulator, an
emulator or a device as you change it; `mygo build` makes what the App
Store and Google Play take.

```sh
mygo dev -platform ios         # the booted iOS simulator
mygo dev -platform android     # the device or emulator adb reaches
mygo build -platform ios       # a signed .ipa
mygo build -platform android   # an .apk and an .aab
```

## What you need

- **iOS**: a Mac with Xcode. The Go code links with Xcode's clang (cgo,
  which Go needs to link for iOS), and the CLI compiles the app's Swift
  host, its icon and its bundle, and signs it, without an Xcode project.
  Devices and the App Store need an Apple developer team and a
  provisioning profile for the app's identifier, which Xcode makes
  (Settings › Accounts) or developer.apple.com.
- **Android**: Android Studio with the SDK and the NDK (SDK Manager › SDK
  Tools › NDK), on any system. The Go code is a shared library for each
  ABI, linked by the NDK's clang, and Gradle builds the app; the CLI finds
  the SDK in its usual place or `ANDROID_HOME`, the NDK in the SDK or
  `ANDROID_NDK_HOME`, and Android Studio's JDK or `JAVA_HOME`.

Desktop builds still need neither cgo nor these tools.

## Developing

`mygo dev -platform ios` builds the app for the simulator, installs it on
the one booted (or `-device`, a simulator's name or UDID, which it boots)
and launches it, the app's output in the console. Each change of the Go
code rebuilds and relaunches it in a few seconds. A frontend's `devUrl`
works as on the desktop: the simulator reaches the Mac's dev server.

`mygo dev -platform android` does the same with the device or emulator
`adb` reaches (or `-device`, its serial), which reaches the dev server
through `adb reverse`, at the same URL. Without `devUrl`, the app embeds
`frontendDist`, as a device cannot read the computer's disk.

The development app is "<name> Dev" with the identifier
"<identifier>.dev", as on the desktop, so it installs beside the release.

## Building

```sh
mygo build -platform ios             # dist/ios-arm64/<slug>-<version>.ipa, signed
mygo build -platform ios/simulator   # dist/ios-simulator/<name>.app
mygo build -platform android         # dist/android/<slug>-<version>.apk and .aab
mygo build -platform android/arm64   # one ABI: arm64, arm or amd64
```

The `.ipa` is signed with the installed provisioning profile that best
fits the app's identifier and `ios.distribution` (`app-store` by default,
which is also TestFlight's), whose certificate is in the keychain; it
carries the privacy manifest App Store Connect asks for, made from what the
app's code uses, unless the project has its own
`ios/PrivacyInfo.xcprivacy`. Upload it with Transporter or `xcrun altool`.
The CLI refuses a build that names Apple's private methods, which App
Review turns down.

The `.aab` is what Google Play takes, the `.apk` what other stores and
devices install. Release builds are signed with `android.keystore`, its
passwords from `MYGO_ANDROID_KEYSTORE_PASSWORD` and
`MYGO_ANDROID_KEY_PASSWORD`, or else with the debug key, which Google Play
refuses.

The configuration's `ios` and `android` sections set the rest (see
[Configuration](configuration.md)):

```ts
export default defineConfig({
  name: "Todo",
  identifier: "dev.example.todo",
  ios: { team: "ABCDE12345", infoPlist: { NSCameraUsageDescription: "Scan receipts." } },
  android: { permissions: ["CAMERA"], keystore: "upload.jks", keyAlias: "upload" },
});
```

## Tabs and stacks

Phones move through an app with the system's bar of tabs at the bottom
and stacks of screens, each with a bar at the top holding its title and a
back button. `mygo.NewTabs` shows a bar of tabs (UIKit's on iOS, Material's
bottom navigation on Android), and `Window.Push` shows a window over
another, on its tab's stack:

```go
mygo.App.WhenReady(func() {
	list := mygo.NewWindow(mygo.WindowOptions{Title: "Fruits", Content: ui.View(app.list), Hidden: true})
	web := mygo.NewWindow(mygo.WindowOptions{Title: "Web", URL: "/", Hidden: true})
	tabs, _ := mygo.NewTabs(
		mygo.TabItem{Title: "Fruits", Symbol: "leaf", Icon: leafPNG, Window: list},
		mygo.TabItem{Title: "Web", Symbol: "globe", Icon: globePNG, Window: web},
	)
	tabs.SetBadge(1, "3")
	app.open = func(fruit string) {
		list.Push(mygo.NewWindow(mygo.WindowOptions{Title: fruit, Content: ui.View(app.detail(fruit)), Hidden: true}))
	}
})
```

A tab names an SF Symbol for iOS and gives a PNG for Android (and for iOS
without a symbol), drawn as a template in the bar's colors. The back
button, iOS's swipe from the left edge and Android's back pop a pushed
window, which closes it; `Window.Pop` does too. On iOS the bars float
over native UI, whose lists scroll on beneath the tab bar's glass; on
Android the bars take the color of the window's background. On desktops
the windows are windows: `Tabs.Select` shows one, and a pushed window
shows as one. `examples/tabs` shows both, with a page in one of its tabs.

## Fingers

Native UI takes fingers as the systems' own toolkits do:

- A tap is a click. A finger reaches what it nearly touches, by 14 DIPs: a
  slider's thumb is narrower than a fingertip.
- A finger that moves locks to the way it went, and to the innermost container
  that scrolls that way: a row of photos in a list down the screen takes what
  goes across, and leaves what goes down to the list. It flings the content on
  as it lifts, and stops a fling as it lands, tapping nothing.
- Past the content's ends, the content follows less and less far and springs
  back, a long way on iOS and less on Android. A scroll indicator shows as the
  content moves.
- What drags along a way takes what goes that way: a sideways `Draggable` in a
  list, even with buttons inside it, leaves what goes down to the list.
- Held still for half a second, a finger long-presses: `Element.LongPressed`,
  a context menu, a word of text, or what drags a value, as a list's rows that
  `Reorder` moves. The device gives a tick.
- Two fingers pinch and turn (`InputEvent`'s `InputZoom`, with `Rotation`).
- From the left edge on iOS, a finger takes a `ui.Router`'s page back, following
  it, as UIKit's stacks do. A tap on iOS's status bar sends what scrolls back to
  its top.

`Element.PullToRefresh` lets a finger pull a scroll container past its top:

```go
list := ui.Scroll(c).Fill()
if list.PullToRefresh(&app.refresh) {
	go func() {
		items := fetch()
		win.Update(func() { app.items, app.refresh.Refreshing = items, false })
	}()
}
```

## Text

The keyboard edits text through its input's `TextInputClient`, with the system's
suggestions and input methods. Elements say what they hold, which chooses the
keyboard:

```go
ui.TextInput(c, &app.email).
	Keyboard(ui.KeyboardEmail).ReturnKey(ui.ReturnNext).
	Capitalize(ui.CapitalizeNone).Autocorrect(false).Autofill(ui.AutofillEmail)
```

`ReturnNext` moves the focus on; `Password` is a secure entry. As the keyboard
comes up it shows the input in use. Holding a word chooses it, with the system's
menu of what to do with it (`UIEditMenuInteraction` and Android's floating
toolbar); the handles at its ends drag them, with the system's magnifier, drawn
as each system's. Tapping the caret shows the menu too.

## Widgets of phones

Package `ui/mobile` is what phones' interfaces are made of and desktops' aren't,
looking as each system's, on `ui`'s public API. Apps that don't import it don't
pay for it.

| | |
|---|---|
| `Sheet`, `ActionSheet` | Panels coming up from the bottom, with detents, which a finger drags away |
| `Drawer` | A panel sliding in from the side |
| `PageView`, `PageDots` | Pages a finger swipes between, built as they come near |
| `SwipeRow` | A row a finger swipes aside to reveal actions |
| `Wheel`, `DatePicker`, `TimePicker` | Wheels that spin, with a tick as the choice changes |
| `TopBar`, `BottomBar`, `FloatingButton` | Bars drawn by the app, for one window that routes within it |
| `ListRow`, `Section` | Rows and groups of them, with a press's look |
| `Ink` | Material's ripple |
| `Switch` | The system's switch |
| `Zoomable` | Content that fingers pinch, turn and drag, as a photo |

`mobile.SetLook` chooses a design whatever the system. `examples/mobile` shows
them all.

## The device

| | |
|---|---|
| `mygo.Haptics` | The feel of a selection, an impact or how a task ended |
| `mygo.SystemBars` | The color of the icons of the system's bars, and hiding them |
| `mygo.SetOrientations` | How the screen may turn |
| `mygo.Share` | The system's share sheet of text, a link and files |
| `mygo.Network` | Whether it is online, and whether that costs |
| `mygo.Power.Battery` | The battery's level, and the low power mode |
| `mygo.SetBarItems` | The buttons of a window's system navigation bar |
| `Context.SafeArea` | What covers the window's edges, to draw in the `Overlay` |

They do nothing on desktops, or report `errors.ErrUnsupported`.

## Official plugins

What needs permissions or a large dependency is a plugin, which an app has only
if it imports it, with its usage descriptions and permissions added to the app:

| Package (`plugins/...`) | What it does |
|---|---|
| `location` | Where the device is, and following it, through Core Location and LocationManager |
| `imagepicker` | The system's picker of photos and videos, which needs no permission, and its camera screen |
| `camera` | A live preview in native UI, and photos: AVCaptureSession and CameraX |
| `videoplayer` | Video in native UI: AVPlayer and MediaPlayer |
| `biometrics` | Face ID, Touch ID, fingerprints, the passcode |
| `sensors` | The accelerometer, the gyroscope and the magnetometer |
| `push` | APNs and Firebase tokens and messages (Firebase on Android needs a project) |
| `preferences` | Small values kept between runs, on every platform |

## Writing a plugin

A plugin with native code is a Go package beside its Swift and Kotlin, which its
`mygo-plugin.json` names. `mygo build` finds it among the packages the app
imports, compiles its code into the host and registers it:

```json
{
  "mobile": {
    "id": "torch",
    "ios": {
      "sources": ["ios/*.swift"],
      "class": "TorchPlugin",
      "frameworks": ["AVFoundation"],
      "infoPlist": {"NSCameraUsageDescription": "Lights the torch."}
    },
    "android": {
      "sources": ["android/*.kt"],
      "class": "TorchPlugin",
      "permissions": ["CAMERA"],
      "dependencies": ["androidx.camera:camera-core:1.6.2"],
      "application": ["<service android:name=\"dev.mygo.TorchService\" />"]
    }
  }
}
```

Go calls it with JSON, and hears its events:

```go
func On(ctx context.Context, on bool) error {
	return mygo.Native("torch").Call(ctx, "set", map[string]bool{"on": on}, nil)
}
```

The Swift goes in the host's module, the Kotlin in package `dev.mygo`, on the UI
thread. They answer once, at any time, and `PluginHost.emit` sends events
(`mygo.Native("torch").On`):

```swift
final class TorchPlugin: MyGoPlugin {
    func call(_ method: String, _ args: [String: Any], _ reply: @escaping PluginReply) {
        switch method {
        case "set": reply(.success(nil))
        default: reply(.failure(PluginFailure("NoMethod", method)))
        }
    }
}
```

```kotlin
package dev.mygo

class TorchPlugin : MyGoPlugin {
    override fun call(method: String, args: JSONObject, reply: PluginReply) {
        when (method) {
            "set" -> reply.success(null)
            else -> reply.failure(PluginFailure("NoMethod", method))
        }
    }
}
```

`PluginHost` has what they need of the host: on iOS the `presenter` to present
over and `applicationEvent` for what reaches the app's delegate; on Android the
`activity`, `startForResult` and `requestPermissions`. An error's code comes to
Go as a `*mygo.NativeError`.

### Native views

`PluginHost.registerView` registers a type of view, in the plugin's initializer,
which `mygo.NewPluginView` makes for a window showing native UI, and `ui.WebView`
places. It shows under the content, in a hole of its frames, as a web view does:
what the view paints after it shows over it, and fingers on that stay the
content's. The view's `ID` names it in calls about it. `plugins/videoplayer` is
a short example, for a map or anything else the system draws.

## Testing

Unit tests run views with a `ui.Tester`, whose clock they advance, with
`TouchDown`, `TouchMove`, `TouchUp` and `SetInsets`. Package `ui/golden` compares
what it draws with images beside the tests:

```go
func TestCard(t *testing.T) {
	golden.Match(t, ui.NewTester(card, 320, 200), "testdata/card.png")
}
```

Run with `MYGO_UPDATE_GOLDEN=1` to write them, on the platform that runs the
tests: text differs from one to another.

`MYGO_FRAME_STATS=1` logs frames slower than 8 ms, with what took the time, also
on phones: `SIMCTL_CHILD_MYGO_FRAME_STATS=1` for a simulator, and on Android
`adb shell am start ... --es mygo.env MYGO_FRAME_STATS=1`.

## How apps behave on phones

- **One screen**: the window shown last fills the screen, above those
  shown before; hiding or closing it shows the one under it. Window sizes,
  positions and title bars have no say. `Window.Bounds` is the screen's.
- **Native UI lays out inside the safe area**: the system's bars, the
  display's cutout and, while it shows, the keyboard, while the window's
  background fills the edges. Pages lay out inside the bars, and above the
  keyboard on Android.
- **Touch** is below ([Fingers](#fingers)); `Tester.TouchDown`, `TouchMove`
  and `TouchUp` test views with a finger.
- **The keyboard** shows while an element of native UI takes text, with
  the system's suggestions and input methods (composition included), and
  a hardware keyboard's keys work as on the desktop ([Text](#text)).
- **Accessibility**: VoiceOver and TalkBack read native UI's elements and
  act on them.
- **Android's back** pops the window pushed last, goes back in a page,
  closes what Escape closes in native UI, goes back in a `ui.Router`, or
  else leaves the app running in the background. The system knows
  beforehand, for its predictive back: a `ui.Router`'s page follows the
  gesture, and without anything to go back to the system shows the app
  leaving.
- **Lifecycle**: the app never quits itself. Going to the background is a
  power event `"suspend"` (`Power.OnSuspend`), and coming back `"resume"`
  and `App.OnActivate`. Frames stop in between. The system may end an app in
  the background: `App.OnSaveState` keeps what it needs to come back, which
  the next launch finds in `App.RestoredState`.
- **Memory**: `App.OnMemoryWarning` is called as the system runs short. Native
  UI lets go of its renderer's textures and its caches of text then, and the
  next frame makes them again.
- **Deep links** (`urlSchemes`) open the app with `App.OnOpenURL`.
- **Dialogs**: message boxes are the system's alerts; `Dialog.Open` shows
  its document picker and gives copies of the files chosen, in the app's
  temporary files. `Dialog.Save` gives a temporary file to write;
  `Dialog.SaveCopy` has the user choose where a file the app made goes.
- **Files**: `WatchFiles` watches the app's directories as on the
  desktop.
- **Secrets** go to the Keychain on iOS, and on Android to the app's
  storage, encrypted with a key of Android's keystore.
- **Notifications** ask for the user's permission the first time.
- What phones lack does nothing: menu bars, trays, the Dock, global
  shortcuts, the updater (stores update apps).

Android's web view loads no custom schemes: the app's frontend
(`mygo://localhost`) and its other schemes are served at
`https://<scheme>.localhost`, which pages never notice but in
`location.href`. Requests of `fetch` and `XMLHttpRequest` to the app's
schemes keep their bodies; a form posted to one arrives without its body
on Android, which does not give it, so post forms with `fetch` or through
bound methods.
