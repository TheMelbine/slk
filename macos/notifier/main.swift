// slk-notifier posts one macOS notification for slk and exits.
//
//   slk-notifier --title T --body B [--image FILE] [--activate BUNDLE_ID]
//                [--focus-title PREFIX]
//
// The notification carries the app bundle's icon, and FILE (the sender's
// avatar) as a thumbnail. A click relaunches the app with no arguments;
// it then brings back the terminal named by --activate, and in Ghostty
// the tab whose title starts with --focus-title.
import AppKit
import UserNotifications

let ghostty = "com.mitchellh.ghostty"

func focusTerminal(bundleID: String, titlePrefix: String) {
    if bundleID == ghostty && !titlePrefix.isEmpty {
        let prefix = titlePrefix.replacingOccurrences(of: "\"", with: "\\\"")
        let src = """
        tell application id "\(ghostty)"
            activate
            focus (first terminal whose name starts with "\(prefix)")
        end tell
        """
        var err: NSDictionary?
        NSAppleScript(source: src)?.executeAndReturnError(&err)
        if err == nil { return }
    }
    if let url = NSWorkspace.shared.urlForApplication(withBundleIdentifier: bundleID) {
        let done = DispatchSemaphore(value: 0)
        NSWorkspace.shared.openApplication(at: url, configuration: .init()) { _, _ in done.signal() }
        _ = done.wait(timeout: .now() + 3)
    }
}

final class Delegate: NSObject, UNUserNotificationCenterDelegate {
    func userNotificationCenter(_ c: UNUserNotificationCenter, willPresent n: UNNotification,
                                withCompletionHandler h: @escaping (UNNotificationPresentationOptions) -> Void) {
        h([.banner, .list, .sound])
    }

    func userNotificationCenter(_ c: UNUserNotificationCenter, didReceive r: UNNotificationResponse,
                                withCompletionHandler h: @escaping () -> Void) {
        let info = r.notification.request.content.userInfo
        if let app = info["activate"] as? String, !app.isEmpty {
            focusTerminal(bundleID: app, titlePrefix: info["focusTitle"] as? String ?? "")
        }
        h()
        exit(0)
    }
}

func fail(_ msg: String) -> Never {
    FileHandle.standardError.write((msg + "\n").data(using: .utf8)!)
    exit(1)
}

var opts: [String: String] = [:]
var it = CommandLine.arguments.dropFirst().makeIterator()
while let key = it.next() {
    guard key.hasPrefix("--"), let value = it.next() else { fail("usage: slk-notifier --title T --body B") }
    opts[String(key.dropFirst(2))] = value
}

let center = UNUserNotificationCenter.current()
let delegate = Delegate()
center.delegate = delegate

if opts["title"] == nil && opts["body"] == nil {
    // Relaunched by a click: wait for didReceive.
    DispatchQueue.main.asyncAfter(deadline: .now() + 5) { exit(0) }
    NSApplication.shared.run()
}

center.requestAuthorization(options: [.alert, .sound]) { granted, err in
    guard granted else { fail("notifications not allowed: \(err?.localizedDescription ?? "denied")") }
    let content = UNMutableNotificationContent()
    content.title = opts["title"] ?? ""
    content.body = opts["body"] ?? ""
    content.sound = .default
    content.userInfo = ["activate": opts["activate"] ?? "", "focusTitle": opts["focus-title"] ?? ""]
    if let image = opts["image"], FileManager.default.fileExists(atPath: image) {
        // UNNotificationAttachment moves the file into its own store, so
        // hand it a copy and leave slk's image cache alone.
        let copy = URL(fileURLWithPath: NSTemporaryDirectory())
            .appendingPathComponent(UUID().uuidString)
            .appendingPathExtension((image as NSString).pathExtension)
        if (try? FileManager.default.copyItem(at: URL(fileURLWithPath: image), to: copy)) != nil,
           let attachment = try? UNNotificationAttachment(identifier: "avatar", url: copy) {
            content.attachments = [attachment]
        }
    }
    let req = UNNotificationRequest(identifier: UUID().uuidString, content: content, trigger: nil)
    center.add(req) { err in
        if let err = err { fail("posting notification: \(err.localizedDescription)") }
        exit(0)
    }
}
NSApplication.shared.run()
