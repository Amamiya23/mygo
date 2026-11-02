import UIKit
import UserNotifications

final class PushPlugin: MyGoPlugin {
    private var waiting: PluginReply?

    func call(_ method: String, _ args: [String: Any], _ reply: @escaping PluginReply) {
        switch method {
        case "configure":
            reply(.success(nil))
        case "register":
            waiting = reply
            UNUserNotificationCenter.current().requestAuthorization(options: [.alert, .sound, .badge]) { granted, error in
                DispatchQueue.main.async {
                    guard granted else {
                        self.waiting?(.failure(PluginFailure("Denied", error?.localizedDescription ?? "notifications are not allowed")))
                        self.waiting = nil
                        return
                    }
                    UIApplication.shared.registerForRemoteNotifications()
                }
            }
        default:
            reply(.failure(PluginFailure("NoMethod", method)))
        }
    }

    func applicationEvent(_ event: ApplicationEvent) {
        switch event {
        case .pushToken(let d):
            let token = d.map { String(format: "%02x", $0) }.joined()
            if let r = waiting {
                r(.success(["token": token]))
                waiting = nil
            }
            PluginHost.emit("push", "token", token)
        case .pushError(let e):
            waiting?(.failure(PluginFailure("Failed", e.localizedDescription)))
            waiting = nil
        case .pushMessage(let info):
            var out: [String: Any] = [:]
            for (k, v) in info where JSONSerialization.isValidJSONObject([v]) { out["\(k)"] = v }
            PluginHost.emit("push", "message", out)
        }
    }
}
